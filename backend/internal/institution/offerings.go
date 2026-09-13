package institution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/audit"
	"github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var schoolCatalogCode = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

var errIdempotencyPayloadConflict = errors.New("idempotency payload conflict")
var errAuthorizationDependencyConflict = errors.New("authorization dependency conflict")

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	buffer := make([]byte, 0, 10)
	for value > 0 {
		buffer = append([]byte{byte('0' + value%10)}, buffer...)
		value /= 10
	}
	return string(buffer)
}

func institutionRequestScope(r *http.Request) (tenantCode, institutionID, actor string, ok bool) {
	tenantCode = strings.TrimSpace(auth.CurrentTenantCodeFromRequest(r))
	institutionID = strings.TrimSpace(auth.CurrentInstitutionIDFromRequest(r))
	actor = strings.TrimSpace(auth.CurrentSubjectFromRequest(r))
	return tenantCode, institutionID, actor, tenantCode != "" && institutionID != "" && actor != ""
}

func decodeClosedJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return false
	}
	return decoder.Decode(&struct{}{}) == io.EOF
}

func normalizeCatalogCode(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func normalizeOptionalDate(value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	return &normalized
}

func validEffectiveWindow(from string, to *string) bool {
	start, err := time.Parse(time.DateOnly, strings.TrimSpace(from))
	if err != nil {
		return false
	}
	to = normalizeOptionalDate(to)
	if to == nil {
		return true
	}
	end, err := time.Parse(time.DateOnly, *to)
	return err == nil && !end.Before(start)
}

func validOptionalDate(value *string) bool {
	value = normalizeOptionalDate(value)
	if value == nil {
		return true
	}
	_, err := time.Parse(time.DateOnly, *value)
	return err == nil
}

func validCatalogLifecycleUpdate(expectedVersion int, label string, active *bool, effectiveTo *string) bool {
	return expectedVersion > 0 && strings.TrimSpace(label) != "" && active != nil && validOptionalDate(effectiveTo) && (*active || normalizeOptionalDate(effectiveTo) != nil)
}

func dateString(value time.Time) string { return value.Format(time.DateOnly) }

func nullableDateString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.DateOnly)
	return &formatted
}

func requestFingerprint(value any) string {
	raw, _ := json.Marshal(value)
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

// reserveInstitutionCreate makes POST replay-safe without accepting scope from
// the browser. The shared idempotency table is RLS-protected and the response
// UUID is reserved before the aggregate insert in the same transaction.
func reserveInstitutionCreate(ctx context.Context, tx pgx.Tx, tenantCode, institutionID, actor, operation, key, fingerprint, proposedID string) (string, bool, error) {
	var responseID string
	err := tx.QueryRow(ctx, `insert into school_operation_idempotency(tenant_code,institution_id,operation,idempotency_key,response_id,request_fingerprint,created_by_subject)
		values($1,$2,$3,$4,$5::uuid,$6,$7) on conflict do nothing returning response_id::text`, tenantCode, institutionID, operation, key, proposedID, fingerprint, actor).Scan(&responseID)
	if err == nil {
		return responseID, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, err
	}
	var storedFingerprint string
	if err = tx.QueryRow(ctx, `select response_id::text,request_fingerprint from school_operation_idempotency
		where tenant_code=$1 and institution_id=$2 and operation=$3 and idempotency_key=$4 for update`, tenantCode, institutionID, operation, key).Scan(&responseID, &storedFingerprint); err != nil {
		return "", false, err
	}
	if storedFingerprint != fingerprint {
		return "", false, errIdempotencyPayloadConflict
	}
	return responseID, true, nil
}

func writeCatalogPersistenceError(w http.ResponseWriter, err error, code string) {
	if errors.Is(err, errIdempotencyPayloadConflict) {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "idempotency_key_payload_conflict"})
		return
	}
	if errors.Is(err, errAuthorizationDependencyConflict) {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": code + "_authorization_dependency_conflict"})
		return
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		if pgError.Code == "23514" && (pgError.ConstraintName == "school_location_authorization_dependency_conflict" || pgError.ConstraintName == "education_offering_authorization_dependency_conflict") {
			httpx.JSON(w, http.StatusConflict, map[string]any{"code": pgError.ConstraintName})
			return
		}
		switch pgError.Code {
		case "23505", "23P01":
			httpx.JSON(w, http.StatusConflict, map[string]any{"code": code + "_conflict"})
			return
		case "22001", "22P02", "23502", "23503", "23514":
			httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": code + "_persist_failed"})
			return
		}
	}
	httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": code + "_persist_failed"})
}

func lockSchoolCatalogEntity(ctx context.Context, tx pgx.Tx, entityKind, id string) error {
	_, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,0))`, "school_catalog:"+entityKind+":"+id)
	return err
}

func authorizationDependencyConflict(ctx context.Context, tx pgx.Tx, tenantCode, institutionID, entityKind, id string, effectiveTo *string) (bool, error) {
	column := ""
	switch entityKind {
	case "location":
		column = "location_id"
	case "offering":
		column = "offering_id"
	default:
		return false, errors.New("unsupported school catalog entity kind")
	}
	var conflict bool
	err := tx.QueryRow(ctx, `select exists(
		select 1 from school_offering_authorizations
		where tenant_code=$1 and institution_id=$2 and `+column+`=$3::uuid
		  and nullif($4,'') is not null
		  and (effective_to is null or effective_to > nullif($4,'')::date)
	)`, tenantCode, institutionID, id, optionalString(effectiveTo)).Scan(&conflict)
	return conflict, err
}

func (s *Service) ListSchoolLocations(w http.ResponseWriter, r *http.Request) {
	tenantCode, institutionID, _, ok := institutionRequestScope(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "institution_context_required"})
		return
	}
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"code": {}, "name": {}, "active": {}, "effective_from": {}}, []string{"code", "name", "active", "effective_from"})
	where := " where tenant_code=$1 and institution_id=$2"
	args := []any{tenantCode, institutionID}
	for _, field := range []string{"code", "name", "active", "effective_from"} {
		value := query.Filters[field]
		if value == "" {
			continue
		}
		args = append(args, value)
		position := itoa(len(args))
		switch field {
		case "code", "name":
			where += " and " + field + " ilike '%' || $" + position + " || '%'"
		case "active":
			if value != "true" && value != "false" {
				httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_location_active_filter"})
				return
			}
			where += " and active=$" + position + "::boolean"
		case "effective_from":
			if !validEffectiveWindow(value, nil) {
				httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_location_date_filter"})
				return
			}
			where += " and effective_from=$" + position + "::date"
		}
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from school_locations"+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_locations_list_failed"})
		return
	}
	sortColumns := map[string]string{"code": "code", "name": "name", "active": "active", "effective_from": "effective_from"}
	sortColumn := "code"
	if query.Sort != "" {
		sortColumn = sortColumns[query.Sort]
	}
	direction := "asc"
	if query.Direction == "desc" {
		direction = "desc"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `select id::text,code,name,address,active,effective_from,effective_to,expected_version from school_locations`+where+" order by "+sortColumn+" "+direction+",id limit $"+itoa(len(args)-1)+" offset $"+itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_locations_list_failed"})
		return
	}
	defer rows.Close()
	items := []SchoolLocation{}
	for rows.Next() {
		var item SchoolLocation
		var from time.Time
		var to *time.Time
		if err = rows.Scan(&item.ID, &item.Code, &item.Name, &item.Address, &item.Active, &from, &to, &item.ExpectedVersion); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_locations_list_failed"})
			return
		}
		item.EffectiveFrom, item.EffectiveTo = dateString(from), nullableDateString(to)
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_locations_list_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) CreateSchoolLocation(w http.ResponseWriter, r *http.Request) {
	tenantCode, institutionID, actor, ok := institutionRequestScope(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "institution_context_required"})
		return
	}
	var input CreateSchoolLocationRequest
	if !decodeClosedJSON(w, r, &input) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_school_location_payload"})
		return
	}
	input.Code = normalizeCatalogCode(input.Code)
	input.Name, input.Address = strings.TrimSpace(input.Name), strings.TrimSpace(input.Address)
	input.EffectiveFrom, input.EffectiveTo, input.IdempotencyKey = strings.TrimSpace(input.EffectiveFrom), normalizeOptionalDate(input.EffectiveTo), strings.TrimSpace(input.IdempotencyKey)
	if !schoolCatalogCode.MatchString(input.Code) || input.Name == "" || input.IdempotencyKey == "" || !validEffectiveWindow(input.EffectiveFrom, input.EffectiveTo) || (!input.Active && input.EffectiveTo == nil) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_school_location_payload"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_location_create_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	id, replay, err := reserveInstitutionCreate(r.Context(), tx, tenantCode, institutionID, actor, "institution.location.create", input.IdempotencyKey, requestFingerprint(input), uuid.NewString())
	if err != nil {
		writeCatalogPersistenceError(w, err, "school_location")
		return
	}
	if !replay {
		_, err = tx.Exec(r.Context(), `insert into school_locations(id,tenant_code,institution_id,code,name,address,active,effective_from,effective_to,created_by_subject,updated_by_subject)
			values($1::uuid,$2,$3,$4,$5,$6,$7,$8::date,nullif($9,'')::date,$10,$10)`, id, tenantCode, institutionID, input.Code, input.Name, input.Address, input.Active, input.EffectiveFrom, optionalString(input.EffectiveTo), actor)
		if err == nil {
			err = audit.Log(r.Context(), tx, audit.Event{ActorSubject: actor, Action: "institution.location.created", TargetType: "school_location", TargetID: id, Summary: "Institution location created"})
		}
	}
	if err != nil {
		writeCatalogPersistenceError(w, err, "school_location")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_location_create_failed"})
		return
	}
	item, err := s.loadSchoolLocation(r.Context(), id)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_location_read_failed"})
		return
	}
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) loadSchoolLocation(ctx context.Context, id string) (SchoolLocation, error) {
	var item SchoolLocation
	var from time.Time
	var to *time.Time
	err := s.pool.QueryRow(ctx, `select id::text,code,name,address,active,effective_from,effective_to,expected_version from school_locations where id=$1::uuid`, id).Scan(&item.ID, &item.Code, &item.Name, &item.Address, &item.Active, &from, &to, &item.ExpectedVersion)
	item.EffectiveFrom, item.EffectiveTo = dateString(from), nullableDateString(to)
	return item, err
}

func (s *Service) UpdateSchoolLocation(w http.ResponseWriter, r *http.Request) {
	tenantCode, institutionID, actor, ok := institutionRequestScope(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "institution_context_required"})
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "locationID"))
	if _, err := uuid.Parse(id); err != nil {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_school_location_id"})
		return
	}
	var input UpdateSchoolLocationRequest
	if !decodeClosedJSON(w, r, &input) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_school_location_payload"})
		return
	}
	input.Name, input.Address, input.EffectiveTo = strings.TrimSpace(input.Name), strings.TrimSpace(input.Address), normalizeOptionalDate(input.EffectiveTo)
	if !validCatalogLifecycleUpdate(input.ExpectedVersion, input.Name, input.Active, input.EffectiveTo) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_school_location_payload"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_location_update_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	if err = lockSchoolCatalogEntity(r.Context(), tx, "location", id); err != nil {
		writeCatalogPersistenceError(w, err, "school_location")
		return
	}
	dependencyConflict, err := authorizationDependencyConflict(r.Context(), tx, tenantCode, institutionID, "location", id, input.EffectiveTo)
	if err != nil {
		writeCatalogPersistenceError(w, err, "school_location")
		return
	}
	if dependencyConflict {
		writeCatalogPersistenceError(w, errAuthorizationDependencyConflict, "school_location")
		return
	}
	result, err := tx.Exec(r.Context(), `update school_locations
		set name=$1,address=$2,active=$3,effective_to=nullif($4,'')::date,
			expected_version=expected_version+1,updated_by_subject=$5,updated_at=now()
		where tenant_code=$6 and institution_id=$7 and id=$8::uuid and expected_version=$9
			and (nullif($4,'') is null or nullif($4,'')::date >= effective_from)
			and (not $3 or nullif($4,'') is null or nullif($4,'')::date >= current_date)`, input.Name, input.Address, *input.Active, optionalString(input.EffectiveTo), actor, tenantCode, institutionID, id, input.ExpectedVersion)
	if err != nil {
		writeCatalogPersistenceError(w, err, "school_location")
		return
	}
	if result.RowsAffected() != 1 {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "school_location_version_or_window_conflict"})
		return
	}
	if err = audit.Log(r.Context(), tx, audit.Event{ActorSubject: actor, Action: "institution.location.updated", TargetType: "school_location", TargetID: id, Summary: "Institution location metadata updated"}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_location_audit_failed"})
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_location_update_failed"})
		return
	}
	item, err := s.loadSchoolLocation(r.Context(), id)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_location_read_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) ListEducationOfferings(w http.ResponseWriter, r *http.Request) {
	tenantCode, institutionID, _, ok := institutionRequestScope(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "institution_context_required"})
		return
	}
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"code": {}, "education_level": {}, "title": {}, "language_code": {}, "active": {}, "effective_from": {}}, []string{"code", "education_level", "title", "language_code", "active", "effective_from"})
	where := " where tenant_code=$1 and institution_id=$2"
	args := []any{tenantCode, institutionID}
	for _, field := range []string{"code", "education_level", "title", "language_code", "active", "effective_from"} {
		value := query.Filters[field]
		if value == "" {
			continue
		}
		args = append(args, value)
		position := itoa(len(args))
		if field == "code" || field == "title" {
			where += " and " + field + " ilike '%' || $" + position + " || '%'"
		} else if field == "active" {
			if value != "true" && value != "false" {
				httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_offering_active_filter"})
				return
			}
			where += " and active=$" + position + "::boolean"
		} else if field == "effective_from" {
			if !validEffectiveWindow(value, nil) {
				httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_offering_date_filter"})
				return
			}
			where += " and effective_from=$" + position + "::date"
		} else {
			where += " and " + field + "=$" + position
		}
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from school_education_offerings"+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_offerings_list_failed"})
		return
	}
	sortColumns := map[string]string{"code": "code", "education_level": "education_level", "title": "title", "language_code": "language_code", "active": "active", "effective_from": "effective_from"}
	sortColumn := "code"
	if query.Sort != "" {
		sortColumn = sortColumns[query.Sort]
	}
	direction := "asc"
	if query.Direction == "desc" {
		direction = "desc"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `select id::text,code,education_level,specialization_code,language_code,title,active,effective_from,effective_to,expected_version from school_education_offerings`+where+" order by "+sortColumn+" "+direction+",id limit $"+itoa(len(args)-1)+" offset $"+itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_offerings_list_failed"})
		return
	}
	defer rows.Close()
	items := []EducationOffering{}
	for rows.Next() {
		var item EducationOffering
		var from time.Time
		var to *time.Time
		if err = rows.Scan(&item.ID, &item.Code, &item.EducationLevel, &item.SpecializationCode, &item.LanguageCode, &item.Title, &item.Active, &from, &to, &item.ExpectedVersion); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_offerings_list_failed"})
			return
		}
		item.EffectiveFrom, item.EffectiveTo = dateString(from), nullableDateString(to)
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_offerings_list_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) CreateEducationOffering(w http.ResponseWriter, r *http.Request) {
	tenantCode, institutionID, actor, ok := institutionRequestScope(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "institution_context_required"})
		return
	}
	var input CreateEducationOfferingRequest
	if !decodeClosedJSON(w, r, &input) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_education_offering_payload"})
		return
	}
	input.Code, input.EducationLevel = normalizeCatalogCode(input.Code), normalizeCatalogCode(input.EducationLevel)
	input.SpecializationCode, input.LanguageCode = normalizeCatalogCode(input.SpecializationCode), strings.ToLower(strings.TrimSpace(input.LanguageCode))
	input.Title, input.EffectiveFrom, input.EffectiveTo, input.IdempotencyKey = strings.TrimSpace(input.Title), strings.TrimSpace(input.EffectiveFrom), normalizeOptionalDate(input.EffectiveTo), strings.TrimSpace(input.IdempotencyKey)
	if input.LanguageCode == "" {
		input.LanguageCode = "ro"
	}
	if !schoolCatalogCode.MatchString(input.Code) || !schoolCatalogCode.MatchString(input.EducationLevel) || (input.SpecializationCode != "" && !schoolCatalogCode.MatchString(input.SpecializationCode)) || !schoolCatalogCode.MatchString(input.LanguageCode) || input.Title == "" || input.IdempotencyKey == "" || !validEffectiveWindow(input.EffectiveFrom, input.EffectiveTo) || (!input.Active && input.EffectiveTo == nil) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_education_offering_payload"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_offering_create_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	id, replay, err := reserveInstitutionCreate(r.Context(), tx, tenantCode, institutionID, actor, "institution.offering.create", input.IdempotencyKey, requestFingerprint(input), uuid.NewString())
	if err != nil {
		writeCatalogPersistenceError(w, err, "education_offering")
		return
	}
	if !replay {
		_, err = tx.Exec(r.Context(), `insert into school_education_offerings(id,tenant_code,institution_id,code,education_level,specialization_code,language_code,title,active,effective_from,effective_to,created_by_subject,updated_by_subject)
			values($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10::date,nullif($11,'')::date,$12,$12)`, id, tenantCode, institutionID, input.Code, input.EducationLevel, input.SpecializationCode, input.LanguageCode, input.Title, input.Active, input.EffectiveFrom, optionalString(input.EffectiveTo), actor)
		if err == nil {
			err = audit.Log(r.Context(), tx, audit.Event{ActorSubject: actor, Action: "institution.education_offering.created", TargetType: "school_education_offering", TargetID: id, Summary: "Education offering created"})
		}
	}
	if err != nil {
		writeCatalogPersistenceError(w, err, "education_offering")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_offering_create_failed"})
		return
	}
	item, err := s.loadEducationOffering(r.Context(), id)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_offering_read_failed"})
		return
	}
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) loadEducationOffering(ctx context.Context, id string) (EducationOffering, error) {
	var item EducationOffering
	var from time.Time
	var to *time.Time
	err := s.pool.QueryRow(ctx, `select id::text,code,education_level,specialization_code,language_code,title,active,effective_from,effective_to,expected_version from school_education_offerings where id=$1::uuid`, id).Scan(&item.ID, &item.Code, &item.EducationLevel, &item.SpecializationCode, &item.LanguageCode, &item.Title, &item.Active, &from, &to, &item.ExpectedVersion)
	item.EffectiveFrom, item.EffectiveTo = dateString(from), nullableDateString(to)
	return item, err
}

func (s *Service) UpdateEducationOffering(w http.ResponseWriter, r *http.Request) {
	tenantCode, institutionID, actor, ok := institutionRequestScope(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "institution_context_required"})
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "offeringID"))
	if _, err := uuid.Parse(id); err != nil {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_education_offering_id"})
		return
	}
	var input UpdateEducationOfferingRequest
	if !decodeClosedJSON(w, r, &input) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_education_offering_payload"})
		return
	}
	input.Title, input.EffectiveTo = strings.TrimSpace(input.Title), normalizeOptionalDate(input.EffectiveTo)
	if !validCatalogLifecycleUpdate(input.ExpectedVersion, input.Title, input.Active, input.EffectiveTo) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_education_offering_payload"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_offering_update_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	if err = lockSchoolCatalogEntity(r.Context(), tx, "offering", id); err != nil {
		writeCatalogPersistenceError(w, err, "education_offering")
		return
	}
	dependencyConflict, err := authorizationDependencyConflict(r.Context(), tx, tenantCode, institutionID, "offering", id, input.EffectiveTo)
	if err != nil {
		writeCatalogPersistenceError(w, err, "education_offering")
		return
	}
	if dependencyConflict {
		writeCatalogPersistenceError(w, errAuthorizationDependencyConflict, "education_offering")
		return
	}
	result, err := tx.Exec(r.Context(), `update school_education_offerings
		set title=$1,active=$2,effective_to=nullif($3,'')::date,
			expected_version=expected_version+1,updated_by_subject=$4,updated_at=now()
		where tenant_code=$5 and institution_id=$6 and id=$7::uuid and expected_version=$8
			and (nullif($3,'') is null or nullif($3,'')::date >= effective_from)
			and (not $2 or nullif($3,'') is null or nullif($3,'')::date >= current_date)`, input.Title, *input.Active, optionalString(input.EffectiveTo), actor, tenantCode, institutionID, id, input.ExpectedVersion)
	if err != nil {
		writeCatalogPersistenceError(w, err, "education_offering")
		return
	}
	if result.RowsAffected() != 1 {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "education_offering_version_or_window_conflict"})
		return
	}
	if err = audit.Log(r.Context(), tx, audit.Event{ActorSubject: actor, Action: "institution.education_offering.updated", TargetType: "school_education_offering", TargetID: id, Summary: "Education offering metadata updated"}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_offering_audit_failed"})
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_offering_update_failed"})
		return
	}
	item, err := s.loadEducationOffering(r.Context(), id)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_offering_read_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func validAuthorizationStatus(value string) bool {
	// "authorized" remains readable as a legacy storage value, but it is not
	// accepted for new legal evidence: Romanian pre-university law defines
	// provisional authorization and accreditation, not a third positive stage.
	_, ok := map[string]struct{}{"provisional": {}, "accredited": {}, "suspended": {}, "withdrawn": {}, "expired": {}}[value]
	return ok
}

func (s *Service) ListOfferingAuthorizations(w http.ResponseWriter, r *http.Request) {
	tenantCode, institutionID, _, ok := institutionRequestScope(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "institution_context_required"})
		return
	}
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"offering_code": {}, "location_code": {}, "status": {}, "decision_reference": {}, "effective_from": {}}, []string{"offering_code", "location_code", "status", "decision_reference", "effective_from"})
	where := " where authz.tenant_code=$1 and authz.institution_id=$2"
	args := []any{tenantCode, institutionID}
	for _, field := range []string{"offering_code", "location_code", "status", "decision_reference", "effective_from"} {
		value := query.Filters[field]
		if value == "" {
			continue
		}
		args = append(args, value)
		position := itoa(len(args))
		switch field {
		case "offering_code":
			where += " and offering.code ilike '%' || $" + position + " || '%'"
		case "location_code":
			where += " and location.code ilike '%' || $" + position + " || '%'"
		case "decision_reference":
			where += " and authz.decision_reference ilike '%' || $" + position + " || '%'"
		case "status":
			where += " and authz.status=$" + position
		case "effective_from":
			if !validEffectiveWindow(value, nil) {
				httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_authorization_date_filter"})
				return
			}
			where += " and authz.effective_from=$" + position + "::date"
		}
	}
	fromClause := ` from school_offering_authorizations authz
		join school_education_offerings offering on offering.tenant_code=authz.tenant_code and offering.institution_id=authz.institution_id and offering.id=authz.offering_id
		join school_locations location on location.tenant_code=authz.tenant_code and location.institution_id=authz.institution_id and location.id=authz.location_id
		join school_regulatory_sources source on source.tenant_code=authz.tenant_code and source.institution_id=authz.institution_id and source.id=authz.source_id`
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*)"+fromClause+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "offering_authorizations_list_failed"})
		return
	}
	sortColumns := map[string]string{"offering_code": "offering.code", "location_code": "location.code", "status": "authz.status", "decision_reference": "authz.decision_reference", "effective_from": "authz.effective_from"}
	sortColumn := "authz.effective_from"
	if query.Sort != "" {
		sortColumn = sortColumns[query.Sort]
	}
	direction := "desc"
	if query.Direction == "asc" {
		direction = "asc"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `select authz.id::text,authz.offering_id::text,offering.code,offering.title,authz.location_id::text,location.code,location.name,authz.status,authz.authority_name,authz.decision_reference,authz.capacity,authz.capacity_unit,authz.shift,authz.effective_from,authz.effective_to,authz.expected_version,source.citation,source.source_url,authz.replaces_authorization_id::text`+fromClause+where+" order by "+sortColumn+" "+direction+",authz.id limit $"+itoa(len(args)-1)+" offset $"+itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "offering_authorizations_list_failed"})
		return
	}
	defer rows.Close()
	items := []OfferingAuthorization{}
	for rows.Next() {
		var item OfferingAuthorization
		var from time.Time
		var to *time.Time
		if err = rows.Scan(&item.ID, &item.OfferingID, &item.OfferingCode, &item.OfferingTitle, &item.LocationID, &item.LocationCode, &item.LocationName, &item.Status, &item.AuthorityName, &item.DecisionReference, &item.Capacity, &item.CapacityUnit, &item.Shift, &from, &to, &item.ExpectedVersion, &item.SourceCitation, &item.SourceURL, &item.ReplacesAuthorizationID); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "offering_authorizations_list_failed"})
			return
		}
		item.EffectiveFrom, item.EffectiveTo = dateString(from), nullableDateString(to)
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "offering_authorizations_list_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) CreateOfferingAuthorization(w http.ResponseWriter, r *http.Request) {
	tenantCode, institutionID, actor, ok := institutionRequestScope(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "institution_context_required"})
		return
	}
	var input CreateOfferingAuthorizationRequest
	if !decodeClosedJSON(w, r, &input) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_offering_authorization_payload"})
		return
	}
	input.OfferingID, input.LocationID, input.Status = strings.TrimSpace(input.OfferingID), strings.TrimSpace(input.LocationID), strings.TrimSpace(input.Status)
	input.AuthorityName, input.DecisionReference = strings.TrimSpace(input.AuthorityName), strings.TrimSpace(input.DecisionReference)
	input.CapacityUnit, input.Shift = strings.TrimSpace(input.CapacityUnit), strings.TrimSpace(input.Shift)
	input.EffectiveFrom, input.EffectiveTo, input.IdempotencyKey = strings.TrimSpace(input.EffectiveFrom), normalizeOptionalDate(input.EffectiveTo), strings.TrimSpace(input.IdempotencyKey)
	if input.ReplacesAuthorizationID != nil {
		value := strings.TrimSpace(*input.ReplacesAuthorizationID)
		input.ReplacesAuthorizationID = &value
	}
	_, offeringErr := uuid.Parse(input.OfferingID)
	_, locationErr := uuid.Parse(input.LocationID)
	_, validCapacityUnit := map[string]struct{}{"students": {}, "study_groups": {}}[input.CapacityUnit]
	_, validShift := map[string]struct{}{"day": {}, "afternoon": {}, "evening": {}}[input.Shift]
	positiveAuthorization := input.Status == "provisional" || input.Status == "accredited"
	if offeringErr != nil || locationErr != nil || !validAuthorizationStatus(input.Status) || input.AuthorityName == "" || input.DecisionReference == "" || input.IdempotencyKey == "" || !validEffectiveWindow(input.EffectiveFrom, input.EffectiveTo) || !validCapacityUnit || !validShift || (input.Capacity != nil && *input.Capacity < 0) || (positiveAuthorization && (input.Capacity == nil || *input.Capacity < 1)) || validateRegulatorySource(input.Source, true) != "" {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_offering_authorization_payload"})
		return
	}
	if (input.ReplacesAuthorizationID == nil) != (input.ExpectedVersion == nil) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "authorization_replacement_version_required"})
		return
	}
	if input.ReplacesAuthorizationID != nil {
		if _, err := uuid.Parse(*input.ReplacesAuthorizationID); err != nil || input.ExpectedVersion == nil || *input.ExpectedVersion < 1 {
			httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_authorization_replacement"})
			return
		}
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "offering_authorization_create_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	if err = lockSchoolCatalogEntity(r.Context(), tx, "offering", input.OfferingID); err != nil {
		writeCatalogPersistenceError(w, err, "offering_authorization")
		return
	}
	if err = lockSchoolCatalogEntity(r.Context(), tx, "location", input.LocationID); err != nil {
		writeCatalogPersistenceError(w, err, "offering_authorization")
		return
	}
	id, replay, err := reserveInstitutionCreate(r.Context(), tx, tenantCode, institutionID, actor, "institution.authorization.create", input.IdempotencyKey, requestFingerprint(input), uuid.NewString())
	if err != nil {
		writeCatalogPersistenceError(w, err, "offering_authorization")
		return
	}
	if !replay {
		if input.ReplacesAuthorizationID != nil {
			var previousOfferingID, previousLocationID string
			var previousFrom time.Time
			err = tx.QueryRow(r.Context(), `select offering_id::text,location_id::text,effective_from from school_offering_authorizations
				where tenant_code=$1 and institution_id=$2 and id=$3::uuid and expected_version=$4 for update`, tenantCode, institutionID, *input.ReplacesAuthorizationID, *input.ExpectedVersion).Scan(&previousOfferingID, &previousLocationID, &previousFrom)
			newFrom, _ := time.Parse(time.DateOnly, input.EffectiveFrom)
			if errors.Is(err, pgx.ErrNoRows) || previousOfferingID != input.OfferingID || previousLocationID != input.LocationID || !newFrom.After(previousFrom) {
				httpx.JSON(w, http.StatusConflict, map[string]any{"code": "authorization_replacement_conflict"})
				return
			}
			if err != nil {
				httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "authorization_replacement_read_failed"})
				return
			}
			result, updateErr := tx.Exec(r.Context(), `update school_offering_authorizations set effective_to=$5::date-1,expected_version=expected_version+1,updated_by_subject=$6,updated_at=now()
				where tenant_code=$1 and institution_id=$2 and id=$3::uuid and expected_version=$4 and (effective_to is null or effective_to >= $5::date)`, tenantCode, institutionID, *input.ReplacesAuthorizationID, *input.ExpectedVersion, input.EffectiveFrom, actor)
			if updateErr != nil || result.RowsAffected() != 1 {
				httpx.JSON(w, http.StatusConflict, map[string]any{"code": "authorization_replacement_conflict"})
				return
			}
		}
		var sourceID string
		now := time.Now().UTC()
		err = tx.QueryRow(r.Context(), `insert into school_regulatory_sources(tenant_code,institution_id,source_kind,citation,article_reference,issuer,source_url,published_on,consolidated_on,checksum_sha256,status,verified_at,verified_by_subject,revalidation_owner_subject,created_by_subject,updated_by_subject)
			values($1,$2,$3,$4,$5,$6,$7,nullif($8,'')::date,nullif($9,'')::date,$10,'active',$11,$12,$12,$12,$12) returning id::text`, tenantCode, institutionID, input.Source.SourceKind, strings.TrimSpace(input.Source.Citation), strings.TrimSpace(input.Source.ArticleReference), strings.TrimSpace(input.Source.Issuer), strings.TrimSpace(input.Source.SourceURL), optionalString(input.Source.PublishedOn), optionalString(input.Source.ConsolidatedOn), strings.TrimSpace(input.Source.ChecksumSHA256), now, actor).Scan(&sourceID)
		if err == nil {
			var inserted pgconn.CommandTag
			inserted, err = tx.Exec(r.Context(), `insert into school_offering_authorizations(id,tenant_code,institution_id,offering_id,location_id,status,authority_name,decision_reference,capacity,capacity_unit,shift,effective_from,effective_to,source_id,replaces_authorization_id,created_by_subject,updated_by_subject)
				select $1::uuid,$2,$3,offering.id,location.id,$6,$7,$8,$9,$10,$11,$12::date,nullif($13,'')::date,$14::uuid,nullif($15,'')::uuid,$16,$16
				from school_education_offerings offering join school_locations location on location.tenant_code=offering.tenant_code and location.institution_id=offering.institution_id
				where offering.tenant_code=$2 and offering.institution_id=$3 and offering.id=$4::uuid
				  and location.id=$5::uuid
				  and offering.effective_from <= $12::date and location.effective_from <= $12::date
				  and (offering.effective_to is null or (nullif($13,'') is not null and nullif($13,'')::date <= offering.effective_to))
				  and (location.effective_to is null or (nullif($13,'') is not null and nullif($13,'')::date <= location.effective_to))`, id, tenantCode, institutionID, input.OfferingID, input.LocationID, input.Status, input.AuthorityName, input.DecisionReference, input.Capacity, input.CapacityUnit, input.Shift, input.EffectiveFrom, optionalString(input.EffectiveTo), sourceID, optionalString(input.ReplacesAuthorizationID), actor)
			if err == nil && inserted.RowsAffected() != 1 {
				httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "authorization_outside_offering_or_location_window"})
				return
			}
		}
		if err == nil {
			err = audit.Log(r.Context(), tx, audit.Event{ActorSubject: actor, Action: "institution.offering_authorization.created", TargetType: "school_offering_authorization", TargetID: id, Summary: "Offering authorization decision appended"})
		}
	}
	if err != nil {
		writeCatalogPersistenceError(w, err, "offering_authorization")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "offering_authorization_create_failed"})
		return
	}
	item, err := s.loadOfferingAuthorization(r.Context(), id)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "offering_authorization_read_failed"})
		return
	}
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) loadOfferingAuthorization(ctx context.Context, id string) (OfferingAuthorization, error) {
	var item OfferingAuthorization
	var from time.Time
	var to *time.Time
	err := s.pool.QueryRow(ctx, `select authz.id::text,authz.offering_id::text,offering.code,offering.title,authz.location_id::text,location.code,location.name,authz.status,authz.authority_name,authz.decision_reference,authz.capacity,authz.capacity_unit,authz.shift,authz.effective_from,authz.effective_to,authz.expected_version,source.citation,source.source_url,authz.replaces_authorization_id::text
		from school_offering_authorizations authz
		join school_education_offerings offering on offering.tenant_code=authz.tenant_code and offering.institution_id=authz.institution_id and offering.id=authz.offering_id
		join school_locations location on location.tenant_code=authz.tenant_code and location.institution_id=authz.institution_id and location.id=authz.location_id
		join school_regulatory_sources source on source.tenant_code=authz.tenant_code and source.institution_id=authz.institution_id and source.id=authz.source_id
		where authz.id=$1::uuid`, id).Scan(&item.ID, &item.OfferingID, &item.OfferingCode, &item.OfferingTitle, &item.LocationID, &item.LocationCode, &item.LocationName, &item.Status, &item.AuthorityName, &item.DecisionReference, &item.Capacity, &item.CapacityUnit, &item.Shift, &from, &to, &item.ExpectedVersion, &item.SourceCitation, &item.SourceURL, &item.ReplacesAuthorizationID)
	item.EffectiveFrom, item.EffectiveTo = dateString(from), nullableDateString(to)
	return item, err
}
