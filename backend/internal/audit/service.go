package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Event struct {
	ActorSubject string
	Action       string
	TargetType   string
	TargetID     string
	Status       string
	Summary      string
	Details      map[string]any
}

type DB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

func Log(ctx context.Context, db DB, event Event) error {
	if db == nil {
		return fmt.Errorf("write audit event: database is not configured")
	}
	if event.ActorSubject == "" {
		event.ActorSubject = "unknown"
	}
	if event.Status == "" {
		event.Status = "success"
	}
	if event.Details == nil {
		event.Details = map[string]any{}
	}

	payload, err := json.Marshal(event.Details)
	if err != nil {
		return fmt.Errorf("marshal audit event details: %w", err)
	}

	// The institution is intentionally read from the PostgreSQL session rather
	// than Event (or request data). This keeps request and worker audit entries
	// bound to the same tenant context enforced by RLS.
	var institutionID string
	if err := db.QueryRow(ctx, `
		select coalesce(nullif(current_setting('app.institution_id', true), ''), '')
	`).Scan(&institutionID); err != nil {
		return fmt.Errorf("read audit institution context: %w", err)
	}
	if strings.TrimSpace(institutionID) == "" {
		return fmt.Errorf("write audit event: missing institution context")
	}

	_, err = db.Exec(ctx, `
		insert into app_audit_log (institution_id, actor_subject, action, target_type, target_id, status, summary, details)
		values (nullif(current_setting('app.institution_id', true), ''), $1, $2, $3, $4, $5, $6, $7::jsonb)
	`, event.ActorSubject, event.Action, event.TargetType, event.TargetID, event.Status, event.Summary, string(payload))
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}
