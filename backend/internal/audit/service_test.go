package audit

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type auditTestRow struct {
	institutionID string
	err           error
}

func (r auditTestRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*string) = r.institutionID
	return nil
}

type auditTestDB struct {
	row      auditTestRow
	execSQL  string
	execArgs []any
	execErr  error
}

func (db *auditTestDB) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (db *auditTestDB) QueryRow(context.Context, string, ...any) pgx.Row        { return db.row }
func (db *auditTestDB) Begin(context.Context) (pgx.Tx, error)                   { return nil, nil }
func (db *auditTestDB) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	db.execSQL = sql
	db.execArgs = args
	return pgconn.NewCommandTag("INSERT 0 1"), db.execErr
}

func TestLogUsesDatabaseInstitutionContext(t *testing.T) {
	db := &auditTestDB{row: auditTestRow{institutionID: "institution-a"}}
	event := Event{ActorSubject: "user-a", Action: "admin.user.update", TargetType: "user", TargetID: "user-a", Summary: "Updated user"}

	if err := Log(context.Background(), db, event); err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	if !strings.Contains(db.execSQL, "current_setting('app.institution_id', true)") {
		t.Fatalf("insert must derive institution from database context: %s", db.execSQL)
	}
	if len(db.execArgs) != 7 {
		t.Fatalf("insert argument count = %d, want 7; institution must not be an event argument", len(db.execArgs))
	}
}

func TestLogRejectsMissingInstitutionContext(t *testing.T) {
	db := &auditTestDB{row: auditTestRow{institutionID: "  "}}

	err := Log(context.Background(), db, Event{Action: "admin.user.update"})
	if err == nil || !strings.Contains(err.Error(), "missing institution context") {
		t.Fatalf("Log() error = %v, want missing institution context", err)
	}
	if db.execSQL != "" {
		t.Fatal("Log() must not insert without an institution context")
	}
}

func TestLogWrapsInstitutionContextFailure(t *testing.T) {
	db := &auditTestDB{row: auditTestRow{err: errors.New("session unavailable")}}

	err := Log(context.Background(), db, Event{Action: "admin.user.update"})
	if err == nil || !strings.Contains(err.Error(), "read audit institution context") {
		t.Fatalf("Log() error = %v, want context read failure", err)
	}
}
