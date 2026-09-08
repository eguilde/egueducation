package auth

import (
	"context"
	"testing"
)

func TestOIDCTenantCodeFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), oidcTenantContextKey{}, " tenant-balotesti ")
	tenantCode, err := oidcTenantCodeFromContext(ctx)
	if err != nil {
		t.Fatalf("resolve bound tenant: %v", err)
	}
	if tenantCode != "tenant-balotesti" {
		t.Fatalf("tenant code = %q, want tenant-balotesti", tenantCode)
	}
}

func TestOIDCTenantCodeFromContextFailsClosed(t *testing.T) {
	for _, ctx := range []context.Context{
		context.Background(),
		context.WithValue(context.Background(), oidcTenantContextKey{}, "  "),
	} {
		if _, err := oidcTenantCodeFromContext(ctx); err == nil {
			t.Fatal("missing tenant binding was accepted")
		}
	}
}
