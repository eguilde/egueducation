package db

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPhoneIdentityReverseProjectionMigration(t *testing.T) {
	const migrationName = "migrations/0092_phone_identity_reverse_projection.sql"
	contents, err := migrationFiles.ReadFile(migrationName)
	if err != nil {
		t.Fatalf("read %s: %v", migrationName, err)
	}
	text := string(contents)
	for _, required := range []string{
		"revoke_profile_phone_from_identity_change",
		"trg_phone_identity_reverse_projection",
		"after delete or update of user_id, identity_type, normalized_value, verified_at, is_primary",
		"phone_number_verified = false",
		"identity_no_longer_primary",
		"verification_revoked",
		"security definer",
		"set search_path = pg_catalog, public",
		"revoke all on function public.revoke_profile_phone_from_identity_change() from public",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("reverse phone projection migration is missing %q", required)
		}
	}
	if strings.Contains(text, "disable row level security") {
		t.Fatal("reverse phone projection migration must not weaken row-level security")
	}
}

func TestPhoneIdentityReverseProjectionPostgres(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("phone reverse-projection PostgreSQL test requires EGUEDUCATION_TEST_DATABASE_URL or TEST_DATABASE_URL")
	}

	ctx := context.Background()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse integration database URL: %v", err)
	}
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("connect integration database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	userID := uuid.New()
	phone := fmt.Sprintf("+409%08d", time.Now().UnixNano()%100000000)
	if _, err := pool.Exec(ctx, `
		insert into app_users (id, sub, name, email, phone_number, phone_number_verified)
		values ($1, $2, 'Phone reverse projection test', $3, '', false)
	`, userID, "phone-reverse-"+userID.String(), "phone-reverse-"+userID.String()+"@example.test"); err != nil {
		t.Fatalf("insert integration user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `delete from app_users where id=$1`, userID)
	})

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin verified phone fixture: %v", err)
	}
	if _, err = tx.Exec(ctx, `
		insert into app_user_identities (user_id, identity_type, normalized_value, display_value, verified_at, is_primary)
		values ($1, 'phone', $2, $2, now(), true)
	`, userID, phone); err == nil {
		_, err = tx.Exec(ctx, `update app_users set phone_number=$2, phone_number_verified=true where id=$1`, userID, phone)
	}
	if err == nil {
		err = tx.Commit(ctx)
	} else {
		_ = tx.Rollback(ctx)
	}
	if err != nil {
		t.Fatalf("seed verified phone fixture: %v", err)
	}

	if _, err := pool.Exec(ctx, `update app_user_identities set verified_at=null where user_id=$1 and identity_type='phone' and is_primary`, userID); err != nil {
		t.Fatalf("de-verify primary phone identity: %v", err)
	}
	assertProjectedPhone(t, ctx, pool, userID, phone, false)

	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin phone re-verification: %v", err)
	}
	if _, err = tx.Exec(ctx, `update app_user_identities set verified_at=now() where user_id=$1 and identity_type='phone' and is_primary`, userID); err == nil {
		_, err = tx.Exec(ctx, `update app_users set phone_number_verified=true where id=$1`, userID)
	}
	if err == nil {
		err = tx.Commit(ctx)
	} else {
		_ = tx.Rollback(ctx)
	}
	if err != nil {
		t.Fatalf("re-verify phone fixture: %v", err)
	}

	if _, err := pool.Exec(ctx, `delete from app_user_identities where user_id=$1 and identity_type='phone' and is_primary`, userID); err != nil {
		t.Fatalf("delete primary phone identity: %v", err)
	}
	assertProjectedPhone(t, ctx, pool, userID, "", false)
}

func assertProjectedPhone(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, wantPhone string, wantVerified bool) {
	t.Helper()
	var phone string
	var verified bool
	if err := pool.QueryRow(ctx, `select phone_number, phone_number_verified from app_users where id=$1`, userID).Scan(&phone, &verified); err != nil {
		t.Fatalf("read phone projection: %v", err)
	}
	if phone != wantPhone || verified != wantVerified {
		t.Fatalf("phone projection=(%q,%t), want (%q,%t)", phone, verified, wantPhone, wantVerified)
	}
}
