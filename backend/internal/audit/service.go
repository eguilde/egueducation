package audit

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
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

func Log(ctx context.Context, pool *pgxpool.Pool, event Event) error {
	if pool == nil {
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

	_, err = pool.Exec(ctx, `
		insert into app_audit_log (actor_subject, action, target_type, target_id, status, summary, details)
		values ($1, $2, $3, $4, $5, $6, $7::jsonb)
	`, event.ActorSubject, event.Action, event.TargetType, event.TargetID, event.Status, event.Summary, string(payload))
	return err
}
