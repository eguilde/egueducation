package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type queryExecer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

type requestConnKey struct{}

// SessionConfig captures the tenant/request identity that should be pinned to a
// PostgreSQL connection for the duration of a request.
type SessionConfig struct {
	TenantID        string
	InstitutionID   string
	InstitutionName string
	TenantSubdomain string
	ActorSubject    string
	IsSuperAdmin    bool
}

type SessionPool struct {
	pool *pgxpool.Pool
}

func NewSessionPool(pool *pgxpool.Pool) *SessionPool {
	return &SessionPool{pool: pool}
}

func (s *SessionPool) Raw() *pgxpool.Pool {
	return s.raw()
}

func (s *SessionPool) raw() *pgxpool.Pool {
	if s == nil {
		return nil
	}
	return s.pool
}

func (s *SessionPool) target(ctx context.Context) queryExecer {
	if conn, ok := requestConnFromContext(ctx); ok {
		return conn
	}
	return s.raw()
}

func (s *SessionPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return s.target(ctx).Query(ctx, sql, args...)
}

func (s *SessionPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return s.target(ctx).QueryRow(ctx, sql, args...)
}

func (s *SessionPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return s.target(ctx).Exec(ctx, sql, args...)
}

func (s *SessionPool) Begin(ctx context.Context) (pgx.Tx, error) {
	return s.target(ctx).Begin(ctx)
}

func requestConnFromContext(ctx context.Context) (*pgxpool.Conn, bool) {
	conn, ok := ctx.Value(requestConnKey{}).(*pgxpool.Conn)
	return conn, ok && conn != nil
}

func WithRequestConn(ctx context.Context, conn *pgxpool.Conn) context.Context {
	return context.WithValue(ctx, requestConnKey{}, conn)
}

func AcquireRequestConn(ctx context.Context, pool *pgxpool.Pool, cfg SessionConfig) (context.Context, func(), error) {
	if pool == nil {
		return ctx, func() {}, nil
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("acquire request connection: %w", err)
	}

	if err := bindRequestSession(ctx, conn, cfg); err != nil {
		conn.Release()
		return nil, nil, err
	}

	nextCtx := WithRequestConn(ctx, conn)
	release := func() {
		_ = clearRequestSession(context.Background(), conn)
		conn.Release()
	}
	return nextCtx, release, nil
}

func bindRequestSession(ctx context.Context, conn *pgxpool.Conn, cfg SessionConfig) error {
	values := map[string]string{
		"app.tenant_id":        strings.TrimSpace(cfg.TenantID),
		"app.institution_id":   strings.TrimSpace(cfg.InstitutionID),
		"app.institution_name": strings.TrimSpace(cfg.InstitutionName),
		"app.tenant_subdomain": strings.TrimSpace(cfg.TenantSubdomain),
		"app.actor_subject":    strings.TrimSpace(cfg.ActorSubject),
		"app.is_super_admin":   fmt.Sprintf("%t", cfg.IsSuperAdmin),
	}

	for key, value := range values {
		if _, err := conn.Exec(ctx, `select set_config($1, $2, false)`, key, value); err != nil {
			return fmt.Errorf("bind request session %s: %w", key, err)
		}
	}
	return nil
}

func clearRequestSession(ctx context.Context, conn *pgxpool.Conn) error {
	for _, key := range []string{
		"app.tenant_id",
		"app.institution_id",
		"app.institution_name",
		"app.tenant_subdomain",
		"app.actor_subject",
		"app.is_super_admin",
	} {
		if _, err := conn.Exec(ctx, `select set_config($1, '', false)`, key); err != nil {
			return fmt.Errorf("clear request session %s: %w", key, err)
		}
	}
	return nil
}
