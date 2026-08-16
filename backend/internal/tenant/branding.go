package tenant

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Branding struct {
	TenantCode    string
	Subdomain     string
	InstitutionID string
	Name          string
	ShortName     string
}

type ResolverOptions struct {
	Environment string
	BaseDomain  string
}
type Resolver struct {
	pool     *pgxpool.Pool
	baseHost string
	dev      bool
}

var configuredResolver struct {
	sync.RWMutex
	value *Resolver
}

func ConfigureResolver(pool *pgxpool.Pool, options ResolverOptions) {
	configuredResolver.Lock()
	defer configuredResolver.Unlock()
	configuredResolver.value = &Resolver{pool: pool, baseHost: normalizeHost(options.BaseDomain), dev: strings.EqualFold(strings.TrimSpace(options.Environment), "development") || strings.EqualFold(strings.TrimSpace(options.Environment), "test")}
}

// ResolveBranding reads the active host directory. Unknown public hosts never
// inherit a default institution; only local development may use a fallback.
func ResolveBranding(host, fallbackName, fallbackInstitutionID string) Branding {
	configuredResolver.RLock()
	resolver := configuredResolver.value
	configuredResolver.RUnlock()
	if resolver != nil {
		if branding, err := resolver.Resolve(context.Background(), host); err == nil {
			return branding
		}
		if !resolver.dev || !IsLocalHost(host) {
			return Branding{}
		}
	}
	// Unit tooling and pure configuration rendering run before main installs the
	// resolver. The server always configures it before opening a listener.
	if resolver == nil {
		name := strings.TrimSpace(fallbackName)
		if name == "" {
			name = "EguEducation"
		}
		institutionID := strings.TrimSpace(fallbackInstitutionID)
		if institutionID == "" {
			institutionID = "inst-001"
		}
		return Branding{TenantCode: DefaultTenantCode(institutionID, ""), Subdomain: "bootstrap", InstitutionID: institutionID, Name: name, ShortName: name}
	}
	if IsLocalHost(host) {
		name := strings.TrimSpace(fallbackName)
		if name == "" {
			name = "EguEducation"
		}
		institutionID := strings.TrimSpace(fallbackInstitutionID)
		if institutionID == "" {
			institutionID = "inst-001"
		}
		return Branding{TenantCode: DefaultTenantCode(institutionID, ""), Subdomain: "local", InstitutionID: institutionID, Name: name, ShortName: name}
	}
	return Branding{}
}

func (r *Resolver) Resolve(ctx context.Context, host string) (Branding, error) {
	if r == nil || r.pool == nil {
		return Branding{}, errors.New("tenant resolver is not configured")
	}
	hostname := normalizeHost(host)
	if hostname == "" {
		return Branding{}, errors.New("hostname is required")
	}
	if IsLocalHost(hostname) && r.dev {
		return Branding{}, errors.New("local host uses development fallback")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Branding{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// The directory lookup precedes authentication; it is isolated in this
	// transaction and does not grant any request a database-wide RLS bypass.
	if _, err = tx.Exec(ctx, `select set_config('app.is_super_admin', 'true', true)`); err != nil {
		return Branding{}, err
	}
	var branding Branding
	err = tx.QueryRow(ctx, `select t.code, t.subdomain, t.institution_id, t.display_name, t.short_name from app_tenant_hostnames h join app_tenants t on t.code=h.tenant_code where h.hostname=$1 and h.active and t.active`, hostname).Scan(&branding.TenantCode, &branding.Subdomain, &branding.InstitutionID, &branding.Name, &branding.ShortName)
	// An explicit inactive (or tenant-disabled) hostname is a deliberate deny.
	// Do not silently turn it back on through the convenience subdomain lookup.
	if errors.Is(err, pgx.ErrNoRows) {
		var configured bool
		if lookupErr := tx.QueryRow(ctx, `select exists(select 1 from app_tenant_hostnames where hostname=$1)`, hostname).Scan(&configured); lookupErr != nil {
			return Branding{}, lookupErr
		} else if configured {
			return Branding{}, pgx.ErrNoRows
		}
	}
	if errors.Is(err, pgx.ErrNoRows) && r.baseHost != "" && strings.HasSuffix(hostname, "."+r.baseHost) {
		subdomain := strings.TrimSuffix(hostname, "."+r.baseHost)
		if subdomain != "" && !strings.Contains(subdomain, ".") {
			err = tx.QueryRow(ctx, `select code, subdomain, institution_id, display_name, short_name from app_tenants where lower(subdomain)=$1 and active`, subdomain).Scan(&branding.TenantCode, &branding.Subdomain, &branding.InstitutionID, &branding.Name, &branding.ShortName)
		}
	}
	if err != nil {
		return Branding{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Branding{}, err
	}
	return branding, nil
}

func DefaultInstitutionID(customerName string) string {
	if strings.Contains(strings.ToLower(strings.TrimSpace(customerName)), "balotesti") {
		return "inst-balotesti"
	}
	return "inst-001"
}

// Deprecated compatibility helper. Request paths should use Branding.TenantCode.
func DefaultTenantCode(institutionID, subdomain string) string {
	switch strings.TrimSpace(strings.ToLower(institutionID)) {
	case "inst-balotesti":
		return "tenant-balotesti"
	case "inst-001":
		return "tenant-egueducation"
	default:
		return ""
	}
}
func normalizeHost(host string) string {
	value := strings.TrimSpace(strings.ToLower(host))
	if value == "" {
		return ""
	}
	value = strings.TrimPrefix(strings.TrimPrefix(value, "https://"), "http://")
	if index := strings.IndexByte(value, '/'); index >= 0 {
		value = value[:index]
	}
	if parsed, _, err := net.SplitHostPort(value); err == nil {
		value = parsed
	}
	return strings.Trim(strings.TrimSuffix(value, "."), "[]")
}
func IsLocalHost(host string) bool {
	hostname := normalizeHost(host)
	return hostname == "" || hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
}
