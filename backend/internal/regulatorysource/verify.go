package regulatorysource

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/eguilde/egueducation/internal/audit"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/jackc/pgx/v5"
)

type verificationCommand struct {
	Tenant          string
	Institution     string
	Actor           string
	SourceID        string
	Key             string
	ExpectedVersion int
	Fingerprint     string
}

var errSourcePermission = errors.New("source permission denied")
var errSourceConflict = errors.New("source command conflict")

func (s *Service) Verify(w http.ResponseWriter, r *http.Request) {
	tenant, institution, actor, ok := sourceScope(r)
	if !ok {
		httpx.JSON(w, 401, map[string]any{"code": "institution_context_required"})
		return
	}
	id, ok := sourceID(w, r)
	if !ok {
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 200 {
		httpx.JSON(w, 422, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in VerifyRegulatorySourceRequest
	if !decode(w, r, &in) || in.ExpectedVersion < 1 {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_regulatory_source_verification"})
		return
	}
	command := verificationCommand{
		Tenant: tenant, Institution: institution, Actor: actor, SourceID: id,
		Key: key, ExpectedVersion: in.ExpectedVersion,
		Fingerprint: fingerprint(struct {
			SourceID string
			Request  VerifyRegulatorySourceRequest
		}{SourceID: id, Request: in}),
	}
	source, replay, err := s.reserveVerification(r.Context(), command)
	if err != nil {
		writeVerificationError(w, err)
		return
	}
	if replay {
		httpx.JSON(w, 200, source)
		return
	}
	if s.fetcher == nil {
		httpx.JSON(w, 503, map[string]any{"code": "publisher_fetcher_unconfigured"})
		return
	}
	if rejectQueryURL(source.PublisherURL) != nil {
		httpx.JSON(w, 422, map[string]any{"code": "unsupported_publisher_url_query"})
		return
	}
	// The reservation is committed; no database transaction spans network I/O.
	evidence, err := s.fetcher.Fetch(r.Context(), source.PublisherURL)
	if err != nil {
		httpx.JSON(w, 422, map[string]any{"code": "verification_fetch_failed"})
		return
	}
	if rejectQueryURL(evidence.URL) != nil {
		httpx.JSON(w, 422, map[string]any{"code": "unsupported_publisher_url_query"})
		return
	}
	verified, err := s.commitVerification(r.Context(), command, evidence)
	if err != nil {
		writeVerificationError(w, err)
		return
	}
	httpx.JSON(w, 200, verified)
}

func writeVerificationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errSourcePermission):
		httpx.JSON(w, 403, map[string]any{"code": "regulatory_source_permission_required"})
	case errors.Is(err, pgx.ErrNoRows):
		httpx.JSON(w, 404, map[string]any{"code": "source_not_found"})
	case errors.Is(err, errSourceConflict):
		httpx.JSON(w, 409, map[string]any{"code": "source_command_conflict"})
	default:
		writeServer(w, "regulatory_source_verification_failed")
	}
}

func (s *Service) verificationTransaction(ctx context.Context, cmd verificationCommand) (pgx.Tx, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	allowed, err := actorHasPermission(ctx, tx, cmd.Actor, "school.regulatory_sources.manage")
	if err != nil || !allowed {
		_ = tx.Rollback(ctx)
		if err != nil {
			return nil, err
		}
		return nil, errSourcePermission
	}
	// Serialize first use as well as retries; a missing row cannot be locked FOR UPDATE.
	_, err = tx.Exec(ctx, "select pg_advisory_xact_lock(hashtextextended($1,0))",
		fingerprint([]string{cmd.Tenant, cmd.Institution, cmd.Actor, "verify", cmd.Key}))
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}
	return tx, nil
}

func verificationRecord(ctx context.Context, tx pgx.Tx, cmd verificationCommand) (*string, []byte, error) {
	var sourceID, fp string
	var evidenceID *string
	var snapshot []byte
	err := tx.QueryRow(ctx,
		"select source_id::text,request_fingerprint,evidence_id::text,source_snapshot from school_regulatory_source_idempotency "+
			"where tenant_code=$1 and institution_id=$2 and actor_subject=$3 and action='verify' and idempotency_key=$4 for update",
		cmd.Tenant, cmd.Institution, cmd.Actor, cmd.Key,
	).Scan(&sourceID, &fp, &evidenceID, &snapshot)
	if err != nil {
		return nil, nil, err
	}
	if sourceID != cmd.SourceID || fp != cmd.Fingerprint {
		return nil, nil, errSourceConflict
	}
	return evidenceID, snapshot, nil
}

func lockedSource(ctx context.Context, tx pgx.Tx, cmd verificationCommand) (RegulatorySource, error) {
	return scanSource(tx.QueryRow(ctx, sourceSelect+
		" where source.tenant_code=$1 and source.institution_id=$2 and source.id=$3::uuid for update of source",
		cmd.Tenant, cmd.Institution, cmd.SourceID,
	))
}

func (s *Service) reserveVerification(ctx context.Context, cmd verificationCommand) (RegulatorySource, bool, error) {
	tx, err := s.verificationTransaction(ctx, cmd)
	if err != nil {
		return RegulatorySource{}, false, err
	}
	defer tx.Rollback(ctx)
	evidenceID, snapshot, recordErr := verificationRecord(ctx, tx, cmd)
	if recordErr != nil && !errors.Is(recordErr, pgx.ErrNoRows) {
		return RegulatorySource{}, false, recordErr
	}
	source, err := lockedSource(ctx, tx, cmd)
	if err != nil {
		return RegulatorySource{}, false, err
	}
	if evidenceID != nil {
		return source, true, tx.Commit(ctx)
	}
	if source.Status != "draft" || source.ExpectedVersion != cmd.ExpectedVersion {
		return RegulatorySource{}, false, errSourceConflict
	}
	if recordErr == nil {
		var original RegulatorySource
		if json.Unmarshal(snapshot, &original) != nil || fingerprint(original) != fingerprint(source) {
			return RegulatorySource{}, false, errSourceConflict
		}
		return source, false, tx.Commit(ctx)
	}
	snapshot, err = json.Marshal(source)
	if err != nil {
		return RegulatorySource{}, false, err
	}
	_, err = tx.Exec(ctx,
		"insert into school_regulatory_source_idempotency "+
			"(tenant_code,institution_id,actor_subject,action,idempotency_key,request_fingerprint,source_id,source_snapshot) "+
			"values($1,$2,$3,'verify',$4,$5,$6::uuid,$7::jsonb)",
		cmd.Tenant, cmd.Institution, cmd.Actor, cmd.Key, cmd.Fingerprint, cmd.SourceID, string(snapshot),
	)
	if err != nil {
		return RegulatorySource{}, false, err
	}
	if err = sourceAudit(ctx, tx, cmd.Actor, cmd.SourceID, "verification_reserved"); err != nil {
		return RegulatorySource{}, false, err
	}
	return source, false, tx.Commit(ctx)
}

func (s *Service) commitVerification(ctx context.Context, cmd verificationCommand, evidence Evidence) (RegulatorySource, error) {
	tx, err := s.verificationTransaction(ctx, cmd)
	if err != nil {
		return RegulatorySource{}, err
	}
	defer tx.Rollback(ctx)
	evidenceID, snapshot, err := verificationRecord(ctx, tx, cmd)
	if err != nil {
		return RegulatorySource{}, err
	}
	source, err := lockedSource(ctx, tx, cmd)
	if err != nil {
		return RegulatorySource{}, err
	}
	if evidenceID != nil {
		return source, tx.Commit(ctx)
	}
	var original RegulatorySource
	if json.Unmarshal(snapshot, &original) != nil || fingerprint(original) != fingerprint(source) {
		return RegulatorySource{}, errSourceConflict
	}
	var newEvidenceID string
	err = tx.QueryRow(ctx,
		"insert into school_regulatory_source_evidence "+
			"(tenant_code,institution_id,source_id,source_version,requested_url,source_snapshot,retrieved_url,content_type,content,sha256,retrieved_at,retrieved_by_subject) "+
			"values($1,$2,$3::uuid,$4,$5,$6::jsonb,$7,$8,$9,$10,$11,$12) returning id::text",
		cmd.Tenant, cmd.Institution, cmd.SourceID, cmd.ExpectedVersion+1, source.PublisherURL, string(snapshot),
		evidence.URL, evidence.ContentType, evidence.Content, evidence.SHA256, evidence.RetrievedAt, cmd.Actor,
	).Scan(&newEvidenceID)
	if err != nil {
		return RegulatorySource{}, err
	}
	_, err = tx.Exec(ctx,
		"update school_regulatory_sources set status='verified',checksum_sha256=$1,verified_at=$2,verified_by_subject=$3,"+
			"revalidation_owner_subject=$3,expected_version=expected_version+1,updated_by_subject=$3,updated_at=now() "+
			"where tenant_code=$4 and institution_id=$5 and id=$6::uuid",
		evidence.SHA256, evidence.RetrievedAt, cmd.Actor, cmd.Tenant, cmd.Institution, cmd.SourceID,
	)
	if err != nil {
		return RegulatorySource{}, err
	}
	_, err = tx.Exec(ctx,
		"update school_regulatory_source_idempotency set evidence_id=$1::uuid "+
			"where tenant_code=$2 and institution_id=$3 and actor_subject=$4 and action='verify' and idempotency_key=$5",
		newEvidenceID, cmd.Tenant, cmd.Institution, cmd.Actor, cmd.Key,
	)
	if err != nil {
		return RegulatorySource{}, err
	}
	if err = sourceAudit(ctx, tx, cmd.Actor, cmd.SourceID, "verified"); err != nil {
		return RegulatorySource{}, err
	}
	result, err := loadSource(tx, ctx, cmd.Tenant, cmd.Institution, cmd.SourceID)
	if err != nil {
		return RegulatorySource{}, err
	}
	return result, tx.Commit(ctx)
}

func sourceAudit(ctx context.Context, tx pgx.Tx, actor, sourceID, action string) error {
	return audit.Log(ctx, tx, audit.Event{
		ActorSubject: actor, Action: "school.regulatory_sources." + action,
		TargetType: "regulatory_source", TargetID: sourceID, Summary: "Regulatory source lifecycle command",
	})
}
