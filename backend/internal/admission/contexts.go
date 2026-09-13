package admission

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/google/uuid"
)

func (s *Service) CreateClassOfferingContext(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "class_offering_context_create_failed")
		return
	}
	key, ok := idempotencyKey(r)
	if !ok {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in CreateClassOfferingContextRequest
	if decode(w, r, &in) != nil || !validUUID(in.ClassID) || !validUUID(in.OfferingID) || !validUUID(in.LocationID) || !validUUID(in.AuthorizationID) || strings.TrimSpace(in.SchoolYear) == "" || !validDate(in.EffectiveFrom) || (in.EffectiveTo != nil && !validDate(*in.EffectiveTo)) || (in.Shift != "day" && in.Shift != "afternoon" && in.Shift != "evening") {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_class_offering_context"})
		return
	}
	if in.EffectiveTo != nil && *in.EffectiveTo < in.EffectiveFrom {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_class_offering_context"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "class_offering_context_create_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionManage); err != nil {
		writeError(w, err, "class_offering_context_create_failed")
		return
	}
	id, replay, err := reserve(r.Context(), tx, sc, "admission.class_offering_context.create", key, fingerprint(in), uuid.NewString())
	if err != nil {
		writeError(w, err, "class_offering_context_create_failed")
		return
	}
	if !replay {
		_, err = tx.Exec(r.Context(), `insert into school_admission_class_offering_contexts(id,tenant_code,institution_id,class_id,offering_id,location_id,authorization_id,school_year,shift,effective_from,effective_to,created_by_subject,updated_by_subject) values($1::uuid,$2,$3,$4::uuid,$5::uuid,$6::uuid,$7::uuid,$8,$9,$10::date,nullif($11,'')::date,$12,$12)`, id, sc.tenant, sc.institution, in.ClassID, in.OfferingID, in.LocationID, in.AuthorizationID, strings.TrimSpace(in.SchoolYear), in.Shift, in.EffectiveFrom, optional(in.EffectiveTo), sc.actor)
		if err == nil {
			err = outbox(r.Context(), tx, sc, "admission_class_offering_context", id, "admission.class_offering_context.created", map[string]any{"class_offering_context_id": id})
		}
		if err == nil {
			err = auditEvent(r.Context(), tx, sc, "admission.class_offering_context.created", "school_admission_class_offering_context", id)
		}
	}
	if err != nil {
		writeError(w, err, "class_offering_context_create_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "class_offering_context_create_failed")
		return
	}
	httpx.JSON(w, 201, CommandResult{ID: id, Status: "active", ExpectedVersion: 1, Replayed: replay})
}

func (s *Service) ListClassOfferingContexts(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "class_offering_context_list_failed")
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "class_offering_context_list_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRead); err != nil {
		writeError(w, err, "class_offering_context_list_failed")
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"school_year": {}, "shift": {}, "effective_from": {}}, []string{"class_id", "school_year", "shift", "active"})
	sort := map[string]string{"school_year": "school_year", "shift": "shift", "effective_from": "effective_from"}[q.Sort]
	if sort == "" {
		sort = "effective_from"
	}
	where := " where tenant_code=$1 and institution_id=$2"
	args := []any{sc.tenant, sc.institution}
	for _, f := range []string{"class_id", "school_year", "shift", "active"} {
		if v := q.Filters[f]; v != "" {
			args = append(args, v)
			n := len(args)
			if f == "class_id" {
				where += fmt.Sprintf(" and class_id=$%d::uuid", n)
			} else if f == "active" {
				where += fmt.Sprintf(" and active=$%d::boolean", n)
			} else {
				where += fmt.Sprintf(" and %s=$%d", f, n)
			}
		}
	}
	var total int
	if err = tx.QueryRow(r.Context(), "select count(*) from school_admission_class_offering_contexts"+where, args...).Scan(&total); err != nil {
		writeError(w, err, "class_offering_context_list_failed")
		return
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := tx.Query(r.Context(), "select id::text,class_id::text,offering_id::text,location_id::text,authorization_id::text,school_year,shift,active,effective_from,effective_to,expected_version from school_admission_class_offering_contexts"+where+fmt.Sprintf(" order by %s %s,id limit $%d offset $%d", sort, q.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		writeError(w, err, "class_offering_context_list_failed")
		return
	}
	defer rows.Close()
	items := []ClassOfferingContext{}
	for rows.Next() {
		var x ClassOfferingContext
		var from time.Time
		var to *time.Time
		if err = rows.Scan(&x.ID, &x.ClassID, &x.OfferingID, &x.LocationID, &x.AuthorizationID, &x.SchoolYear, &x.Shift, &x.Active, &from, &to, &x.ExpectedVersion); err != nil {
			writeError(w, err, "class_offering_context_list_failed")
			return
		}
		x.EffectiveFrom = from.Format(time.DateOnly)
		if to != nil {
			v := to.Format(time.DateOnly)
			x.EffectiveTo = &v
		}
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		writeError(w, err, "class_offering_context_list_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "class_offering_context_list_failed")
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}
