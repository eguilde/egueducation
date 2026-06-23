package audit

import (
	"context"
	"encoding/json"

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
		return nil
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
		return err
	}

	_, err = db.Exec(ctx, `
		insert into app_audit_log (actor_subject, action, target_type, target_id, status, summary, details)
		values ($1, $2, $3, $4, $5, $6, $7::jsonb)
	`, event.ActorSubject, event.Action, event.TargetType, event.TargetID, event.Status, event.Summary, string(payload))
	return err
}
