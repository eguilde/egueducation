package tenant

import (
	"net"
	"strings"
)

type Branding struct {
	Subdomain     string
	InstitutionID string
	Name          string
	ShortName     string
}

func DefaultInstitutionID(customerName string) string {
	value := strings.ToLower(strings.TrimSpace(customerName))
	switch {
	case strings.Contains(value, "balotesti"), strings.Contains(value, "balotești"), strings.Contains(value, "scoalabalotesti"):
		return "inst-balotesti"
	default:
		return "inst-001"
	}
}

func DefaultTenantCode(institutionID, subdomain string) string {
	switch strings.TrimSpace(strings.ToLower(institutionID)) {
	case "inst-balotesti":
		return "tenant-balotesti"
	case "inst-001", "":
		if strings.Contains(strings.ToLower(strings.TrimSpace(subdomain)), "balotesti") {
			return "tenant-balotesti"
		}
		return "tenant-egueducation"
	default:
		suffix := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(institutionID)), "inst-")
		if suffix == "" {
			if strings.Contains(strings.ToLower(strings.TrimSpace(subdomain)), "balotesti") {
				return "tenant-balotesti"
			}
			return "tenant-egueducation"
		}
		return "tenant-" + suffix
	}
}

func ResolveBranding(host, fallbackName, fallbackInstitutionID string) Branding {
	hostname := normalizeHost(host)
	switch {
	case strings.Contains(hostname, "scoalabalotesti"):
		return Branding{
			Subdomain:     "scoalabalotesti",
			InstitutionID: "inst-balotesti",
			Name:          "Școala Gimnazială nr. 1 Balotești",
			ShortName:     "Balotești",
		}
	case strings.Contains(hostname, "egueducation"):
		return Branding{
			Subdomain:     "egueducation",
			InstitutionID: "inst-001",
			Name:          "EguEducation",
			ShortName:     "EguEducation",
		}
	case hostname != "" && hostname != "localhost" && hostname != "127.0.0.1" && hostname != "::1":
		label := firstLabel(hostname)
		return Branding{
			Subdomain:     label,
			InstitutionID: fallbackInstitutionID,
			Name:          titleCase(label),
			ShortName:     titleCase(label),
		}
	default:
		value := strings.TrimSpace(fallbackName)
		if value == "" {
			value = "EguEducation"
		}
		return Branding{
			Subdomain:     "egueducation",
			InstitutionID: fallbackInstitutionID,
			Name:          value,
			ShortName:     value,
		}
	}
}

func normalizeHost(host string) string {
	value := strings.TrimSpace(strings.ToLower(host))
	if value == "" {
		return ""
	}
	if parsed, _, err := net.SplitHostPort(value); err == nil {
		value = parsed
	}
	if strings.Contains(value, ":") && !strings.Contains(value, "]") {
		parts := strings.Split(value, ":")
		value = parts[0]
	}
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	if idx := strings.IndexByte(value, '/'); idx >= 0 {
		value = value[:idx]
	}
	return value
}

func IsLocalHost(host string) bool {
	hostname := normalizeHost(host)
	return hostname == "" || hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
}

func firstLabel(hostname string) string {
	parts := strings.Split(hostname, ".")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "www" || part == "app" {
			continue
		}
		return part
	}
	return hostname
}

func titleCase(value string) string {
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.ReplaceAll(value, "_", " ")
	parts := strings.Fields(value)
	for index, part := range parts {
		if len(part) == 0 {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}
