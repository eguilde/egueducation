package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/config"
	"github.com/eguilde/egueducation/internal/dossier"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	cfg  config.Config
	pool *pgxpool.Pool
}

func NewService(cfg config.Config, pool *pgxpool.Pool) *Service {
	return &Service{
		cfg:  cfg,
		pool: pool,
	}
}

func (s *Service) Dashboard(w http.ResponseWriter, _ *http.Request) {
	readyDossiers, blockedDossiers := s.readinessStats()
	users := s.scalarCount("select count(*) from app_users")
	memberships := s.scalarCount("select count(*) from app_memberships")
	positions := s.scalarCount("select count(*) from app_positions")
	permissions := s.scalarCount("select count(*) from app_permissions")
	workflows := s.scalarCount("select count(*) from workflow_definitions")
	archives := s.scalarCount("select count(*) from archive_records where institution_id = 'inst-001'")
	httpx.JSON(w, http.StatusOK, DashboardResponse{
		Stats: DashboardStats{
			Users:           users,
			Memberships:     memberships,
			Positions:       positions,
			Permissions:     permissions,
			Workflows:       workflows,
			Archives:        archives,
			ReadyDossiers:   readyDossiers,
			BlockedDossiers: blockedDossiers,
		},
		Modules: []ModuleStatus{
			{Code: "registratura", Active: true},
			{Code: "workflow", Active: true},
			{Code: "earchiva", Active: true},
			{Code: "education", Active: true},
			{Code: "gdpr", Active: s.cfg.EnableGDPRFeatures},
		},
		AdminSections: []string{
			"users",
			"roles",
			"dossier_requirements",
			"memberships",
			"org_units",
			"positions",
			"permissions",
			"workflow_definitions",
			"registry_nomenclatures",
			"archive_nomenclatures",
			"education_taxonomies",
			"gdpr_policies",
			"gdpr_settings",
			"auth_methods",
			"audit",
		},
		Warnings: []string{
			"OIDC provider, workflow runtime, archive ingestion, and education domain services are scaffolded but not yet fully ported.",
		},
	})
}

func (s *Service) logAudit(r *http.Request, action string, targetType string, targetID string, summary string, details map[string]any) {
	_ = audit.Log(r.Context(), s.pool, audit.Event{
		ActorSubject: authruntime.CurrentSubjectFromRequest(r),
		Action:       action,
		TargetType:   targetType,
		TargetID:     targetID,
		Summary:      summary,
		Details:      details,
	})
}

func (s *Service) scalarCount(sql string) int {
	var count int
	if s.pool == nil {
		return 0
	}
	if err := s.pool.QueryRow(context.Background(), sql).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Service) readinessStats() (int, int) {
	if s.pool == nil {
		return 0, 0
	}

	rows, err := s.pool.Query(context.Background(), `
		select
			wi.source_module,
			case when wi.source_record_id is null then null else wi.source_record_id::text end as source_record_id,
			`+dossier.CountSQL("link_stats")+`
		from workflow_instances wi
		`+dossier.LateralJoinSQL("link_stats", "wi.source_module", "wi.source_record_id")+`
		where wi.institution_id = 'inst-001'
			and wi.status <> 'archived'
	`)
	if err != nil {
		return 0, 0
	}
	defer rows.Close()

	ready := 0
	blocked := 0
	for rows.Next() {
		var sourceModule string
		var sourceRecordID *string
		var counts dossier.RelationCounts
		if err := rows.Scan(
			&sourceModule,
			&sourceRecordID,
			&counts.Total,
			&counts.Primary,
			&counts.Supporting,
			&counts.Decision,
			&counts.ArchiveBasis,
			&counts.GDPRBasis,
		); err != nil {
			return 0, 0
		}
		dossierReady, _, err := dossier.Evaluate(context.Background(), s.pool, sourceModule, sourceRecordID != nil, counts, dossier.PurposeReadiness)
		if err != nil {
			return 0, 0
		}
		if dossierReady {
			ready++
		} else {
			blocked++
		}
	}
	if err := rows.Err(); err != nil {
		return 0, 0
	}
	return ready, blocked
}

func (s *Service) ListUsers(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"name":          {},
			"email":         {},
			"position":      {},
			"status":        {},
			"locale":        {},
			"last_login_at": {},
		},
		[]string{"name", "email", "position", "status", "locale"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, column+" like $"+strconv.Itoa(len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, column+" = $"+strconv.Itoa(len(args)))
	}

	if value := strings.TrimSpace(query.Filters["name"]); value != "" {
		addContains("lower(u.name)", value)
	}
	if value := strings.TrimSpace(query.Filters["email"]); value != "" {
		addContains("lower(u.email)", value)
	}
	if value := strings.TrimSpace(query.Filters["position"]); value != "" {
		addEqual("coalesce(pp.name, '')", value)
	}
	if value := strings.TrimSpace(query.Filters["status"]); value != "" {
		addEqual("u.status", value)
	}
	if value := strings.TrimSpace(query.Filters["locale"]); value != "" {
		addEqual("u.locale", value)
	}

	whereClause := "where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), `
		select count(*)
		from app_users u
		left join app_memberships pm on pm.user_id = u.id and pm.is_primary = true
		left join app_positions pp on pp.code = pm.position_code
		`+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_users_failed"})
		return
	}

	sortField := "u.name"
	switch query.Sort {
	case "email":
		sortField = "u.email"
	case "position":
		sortField = "coalesce(pp.name, '')"
	case "status":
		sortField = "u.status"
	case "locale":
		sortField = "u.locale"
	case "last_login_at":
		sortField = "u.last_login_at"
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			u.id::text,
			u.sub,
			u.name,
			u.email,
			u.phone_number,
			coalesce(pp.name, ''),
			u.locale,
			u.status,
			u.email_verified,
			u.phone_number_verified,
			u.preferred_otp_channel,
			coalesce(to_char(u.last_login_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		from app_users u
		left join app_memberships pm on pm.user_id = u.id and pm.is_primary = true
		left join app_positions pp on pp.code = pm.position_code
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, u.name
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_users_failed"})
		return
	}
	defer rows.Close()

	items := []AdminUser{}
	for rows.Next() {
		var item AdminUser
		if err := rows.Scan(&item.ID, &item.Sub, &item.Name, &item.Email, &item.Phone, &item.Position, &item.Locale, &item.Status, &item.EmailVerified, &item.PhoneVerified, &item.PreferredOTPChannel, &item.LastLoginAt); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_users_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_users_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) UpsertUser(w http.ResponseWriter, r *http.Request) {
	var req UpsertUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_user"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Phone = strings.TrimSpace(req.Phone)
	req.Locale = strings.TrimSpace(req.Locale)
	req.Status = strings.TrimSpace(req.Status)
	req.PreferredOTPChannel = strings.TrimSpace(req.PreferredOTPChannel)
	if req.Name == "" || req.Email == "" || req.Locale == "" || req.Status == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_user"})
		return
	}
	if req.PreferredOTPChannel == "" {
		req.PreferredOTPChannel = "sms"
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_user_save_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	subject := req.Email
	var item AdminUser
	if req.ID == "" {
		err = tx.QueryRow(r.Context(), `
			insert into app_users (
				sub, name, email, phone_number, locale, status,
				email_verified, phone_number_verified, preferred_otp_channel
			) values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			returning id::text, sub, name, email, phone_number, locale, status, email_verified, phone_number_verified, preferred_otp_channel
		`, subject, req.Name, req.Email, req.Phone, req.Locale, req.Status, req.EmailVerified, req.PhoneVerified, req.PreferredOTPChannel).Scan(
			&item.ID, &item.Sub, &item.Name, &item.Email, &item.Phone, &item.Locale, &item.Status, &item.EmailVerified, &item.PhoneVerified, &item.PreferredOTPChannel,
		)
	} else {
		err = tx.QueryRow(r.Context(), `
			update app_users
			set sub = $2,
				name = $3,
				email = $4,
				phone_number = $5,
				locale = $6,
				status = $7,
				email_verified = $8,
				phone_number_verified = $9,
				preferred_otp_channel = $10,
				updated_at = now()
			where id::text = $1
			returning id::text, sub, name, email, phone_number, locale, status, email_verified, phone_number_verified, preferred_otp_channel
		`, req.ID, subject, req.Name, req.Email, req.Phone, req.Locale, req.Status, req.EmailVerified, req.PhoneVerified, req.PreferredOTPChannel).Scan(
			&item.ID, &item.Sub, &item.Name, &item.Email, &item.Phone, &item.Locale, &item.Status, &item.EmailVerified, &item.PhoneVerified, &item.PreferredOTPChannel,
		)
	}
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_user_save_failed"})
		return
	}

	if _, err := tx.Exec(r.Context(), `
		insert into app_session_context(user_id, institution_id, institution_name, auth_methods, gdpr_capabilities)
		values ($1::uuid, 'inst-001', 'Colegiul Național EguEducation', array['oidc_redirect', 'sms_otp', 'passkey', 'eudi_wallet'], array['retention_policies', 'subject_export', 'purpose_limited_access', 'publication_anonymization'])
		on conflict (user_id) do nothing
	`, item.ID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_user_save_failed"})
		return
	}

	if _, err := tx.Exec(r.Context(), `
		insert into app_user_modules(user_id, module_code)
		select $1::uuid, code
		from app_modules
		where active = true
		on conflict do nothing
	`, item.ID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_user_save_failed"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_user_save_failed"})
		return
	}

	s.logAudit(r, "admin.users.upsert", "user", item.ID, "Saved administrative user", map[string]any{
		"email":  item.Email,
		"status": item.Status,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UserFilters(w http.ResponseWriter, _ *http.Request) {
	load := func(sql string) ([]string, error) {
		rows, err := s.pool.Query(context.Background(), sql)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		values := []string{}
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, rows.Err()
	}
	positions, err := load(`select distinct coalesce(pp.name, '') from app_users u left join app_memberships pm on pm.user_id = u.id and pm.is_primary = true left join app_positions pp on pp.code = pm.position_code where coalesce(pp.name, '') <> '' order by 1`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_user_filters_failed"})
		return
	}
	statuses, err := load(`select distinct status from app_users order by status`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_user_filters_failed"})
		return
	}
	locales, err := load(`select distinct locale from app_users order by locale`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_user_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"positions": positions,
		"statuses":  statuses,
		"locales":   locales,
	})
}

func (s *Service) ListRoles(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{"code": {}, "label": {}},
		[]string{"code", "label"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["code"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(code) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["label"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(label) like $"+strconv.Itoa(len(args)))
	}

	whereClause := "where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), `select count(*) from app_roles `+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_roles_failed"})
		return
	}

	sortField := "code"
	if query.Sort == "label" {
		sortField = "label"
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select code, label
		from app_roles
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, code
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_roles_failed"})
		return
	}
	defer rows.Close()

	items := []Role{}
	for rows.Next() {
		var item Role
		if err := rows.Scan(&item.Code, &item.Label); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_roles_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_roles_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) UpsertRole(w http.ResponseWriter, r *http.Request) {
	var req UpsertRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_role"})
		return
	}

	req.Code = strings.TrimSpace(req.Code)
	req.Label = strings.TrimSpace(req.Label)
	if req.Code == "" || req.Label == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_role"})
		return
	}

	var item Role
	if err := s.pool.QueryRow(r.Context(), `
		insert into app_roles (code, label)
		values ($1, $2)
		on conflict (code) do update
		set label = excluded.label
		returning code, label
	`, req.Code, req.Label).Scan(&item.Code, &item.Label); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_role_save_failed"})
		return
	}

	s.logAudit(r, "admin.roles.upsert", "role", item.Code, "Saved administrative role", map[string]any{
		"label": item.Label,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) ListUserRoleAssignments(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{"user_name": {}, "role_code": {}},
		[]string{"user_name", "role_code"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["user_name"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "(lower(u.name) like $"+strconv.Itoa(len(args))+" or lower(u.email) like $"+strconv.Itoa(len(args))+")")
	}
	if value := strings.TrimSpace(query.Filters["role_code"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "ur.role_code = $"+strconv.Itoa(len(args)))
	}

	whereClause := "where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), `
		select count(*)
		from app_user_roles ur
		join app_users u on u.id = ur.user_id
		join app_roles r on r.code = ur.role_code
		`+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_role_assignments_failed"})
		return
	}

	sortField := "u.name"
	if query.Sort == "role_code" {
		sortField = "ur.role_code"
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			u.id::text || ':' || ur.role_code as id,
			u.id::text,
			u.name,
			u.email,
			ur.role_code,
			r.label
		from app_user_roles ur
		join app_users u on u.id = ur.user_id
		join app_roles r on r.code = ur.role_code
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, ur.role_code
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_role_assignments_failed"})
		return
	}
	defer rows.Close()

	items := []UserRoleAssignment{}
	for rows.Next() {
		var item UserRoleAssignment
		if err := rows.Scan(&item.ID, &item.UserID, &item.UserName, &item.UserEmail, &item.RoleCode, &item.RoleLabel); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_role_assignments_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_role_assignments_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) UpsertUserRoleAssignment(w http.ResponseWriter, r *http.Request) {
	var req UpsertUserRoleAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_role_assignment"})
		return
	}

	req.UserID = strings.TrimSpace(req.UserID)
	req.RoleCode = strings.TrimSpace(req.RoleCode)
	if req.UserID == "" || req.RoleCode == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_role_assignment"})
		return
	}

	if req.Assigned {
		if _, err := s.pool.Exec(r.Context(), `
			insert into app_user_roles (user_id, role_code)
			values ($1::uuid, $2)
			on conflict do nothing
		`, req.UserID, req.RoleCode); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_role_assignment_save_failed"})
			return
		}
	} else {
		if _, err := s.pool.Exec(r.Context(), `
			delete from app_user_roles
			where user_id = $1::uuid and role_code = $2
		`, req.UserID, req.RoleCode); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_role_assignment_save_failed"})
			return
		}
	}

	var item UserRoleAssignment
	if err := s.pool.QueryRow(r.Context(), `
		select
			u.id::text || ':' || r.code as id,
			u.id::text,
			u.name,
			u.email,
			r.code,
			r.label
		from app_users u
		join app_roles r on r.code = $2
		where u.id = $1::uuid
	`, req.UserID, req.RoleCode).Scan(&item.ID, &item.UserID, &item.UserName, &item.UserEmail, &item.RoleCode, &item.RoleLabel); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_role_assignment_save_failed"})
		return
	}

	s.logAudit(r, "admin.roles.assignment", "user_role", item.ID, "Updated user role assignment", map[string]any{
		"user_id":   item.UserID,
		"role_code": item.RoleCode,
		"assigned":  req.Assigned,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) ListAuditEvents(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"actor_subject": {},
			"domain":        {},
			"action":        {},
			"target_type":   {},
			"status":        {},
			"created_at":    {},
		},
		[]string{"actor_subject", "domain", "action", "target_type", "status", "created_at"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["actor_subject"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(actor_subject) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["domain"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "split_part(action, '.', 1) = $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["action"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(action) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["target_type"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(target_type) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["status"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "status = $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["created_at"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "created_at::date = $"+strconv.Itoa(len(args))+"::date")
	}

	whereClause := "where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), `select count(*) from app_audit_log `+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_audit_failed"})
		return
	}

	sortField := "created_at"
	switch query.Sort {
	case "actor_subject":
		sortField = "actor_subject"
	case "domain":
		sortField = "split_part(action, '.', 1)"
	case "action":
		sortField = "action"
	case "target_type":
		sortField = "target_type"
	case "status":
		sortField = "status"
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			id::text,
			actor_subject,
			split_part(action, '.', 1) as domain,
			action,
			target_type,
			target_id,
			status,
			summary,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		from app_audit_log
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, created_at desc
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_audit_failed"})
		return
	}
	defer rows.Close()

	items := []AuditEvent{}
	for rows.Next() {
		var item AuditEvent
		if err := rows.Scan(&item.ID, &item.ActorSubject, &item.Domain, &item.Action, &item.TargetType, &item.TargetID, &item.Status, &item.Summary, &item.CreatedAt); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_audit_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_audit_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) AuditFilters(w http.ResponseWriter, r *http.Request) {
	load := func(sql string) ([]string, error) {
		rows, err := s.pool.Query(r.Context(), sql)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		values := []string{}
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, rows.Err()
	}

	domains, err := load(`select distinct split_part(action, '.', 1) as domain from app_audit_log order by domain`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_audit_filters_failed"})
		return
	}
	targetTypes, err := load(`select distinct target_type from app_audit_log order by target_type`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_audit_filters_failed"})
		return
	}
	statuses, err := load(`select distinct status from app_audit_log order by status`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_audit_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"domains":      domains,
		"target_types": targetTypes,
		"statuses":     statuses,
	})
}

func (s *Service) ListMemberships(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"user_name":         {},
			"position_name":     {},
			"organization_name": {},
			"org_unit_code":     {},
			"active":            {},
			"is_primary":        {},
			"start_date":        {},
		},
		[]string{"user_name", "position_name", "organization_name", "start_date"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["user_name"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(u.name) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["position_name"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(p.name) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["organization_name"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(coalesce(ou.name, m.organization_name)) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["org_unit_code"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "m.org_unit_code = $"+strconv.Itoa(len(args)))
	}
	for _, filter := range []struct {
		key    string
		column string
	}{
		{"active", "m.active"},
		{"is_primary", "m.is_primary"},
	} {
		if value := strings.TrimSpace(query.Filters[filter.key]); value != "" {
			args = append(args, value == "true")
			clauses = append(clauses, filter.column+" = $"+strconv.Itoa(len(args)))
		}
	}
	if value := strings.TrimSpace(query.Filters["start_date"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "m.start_date = $"+strconv.Itoa(len(args))+"::date")
	}

	whereClause := "where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), `
		select count(*)
		from app_memberships m
		join app_users u on u.id = m.user_id
		join app_positions p on p.code = m.position_code
		join app_org_units ou on ou.code = m.org_unit_code
		`+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_memberships_failed"})
		return
	}

	sortField := "u.name"
	switch query.Sort {
	case "position_name":
		sortField = "p.name"
	case "organization_name":
		sortField = "coalesce(ou.name, m.organization_name)"
	case "start_date":
		sortField = "m.start_date"
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			m.id::text,
			u.id::text,
			u.name,
			u.email,
			m.position_code,
			p.name,
			m.org_unit_code,
			coalesce(ou.name, m.organization_name),
			m.is_primary,
			m.active,
			to_char(m.start_date, 'YYYY-MM-DD'),
			coalesce(to_char(m.end_date, 'YYYY-MM-DD'), '')
		from app_memberships m
		join app_users u on u.id = m.user_id
		join app_positions p on p.code = m.position_code
		join app_org_units ou on ou.code = m.org_unit_code
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, u.name
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_memberships_failed"})
		return
	}
	defer rows.Close()

	items := []Membership{}
	for rows.Next() {
		var item Membership
		if err := rows.Scan(&item.ID, &item.UserID, &item.UserName, &item.UserEmail, &item.PositionCode, &item.PositionName, &item.OrgUnitCode, &item.OrganizationName, &item.IsPrimary, &item.Active, &item.StartDate, &item.EndDate); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_memberships_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_memberships_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) UpsertMembership(w http.ResponseWriter, r *http.Request) {
	var req UpsertMembershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_membership"})
		return
	}

	req.UserID = strings.TrimSpace(req.UserID)
	req.PositionCode = strings.TrimSpace(req.PositionCode)
	req.OrgUnitCode = strings.TrimSpace(req.OrgUnitCode)
	req.OrganizationName = strings.TrimSpace(req.OrganizationName)
	req.StartDate = strings.TrimSpace(req.StartDate)
	req.EndDate = strings.TrimSpace(req.EndDate)
	if req.UserID == "" || req.PositionCode == "" || req.OrgUnitCode == "" || req.StartDate == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_membership"})
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_membership_save_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	if err := tx.QueryRow(r.Context(), `select name from app_org_units where code = $1 and active = true`, req.OrgUnitCode).Scan(&req.OrganizationName); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_membership"})
		return
	}

	if req.IsPrimary {
		if _, err := tx.Exec(r.Context(), `update app_memberships set is_primary = false, updated_at = now() where user_id::text = $1`, req.UserID); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_membership_save_failed"})
			return
		}
	}

	var item Membership
	if req.ID == "" {
		err = tx.QueryRow(r.Context(), `
			insert into app_memberships (user_id, position_code, org_unit_code, organization_name, is_primary, active, start_date, end_date)
			select $1::uuid, $2, $3, $4, $5, $6, $7::date, nullif($8, '')::date
			returning id::text
		`, req.UserID, req.PositionCode, req.OrgUnitCode, req.OrganizationName, req.IsPrimary, req.Active, req.StartDate, req.EndDate).Scan(&item.ID)
	} else {
		err = tx.QueryRow(r.Context(), `
			update app_memberships
			set user_id = $2::uuid,
				position_code = $3,
				org_unit_code = $4,
				organization_name = $5,
				is_primary = $6,
				active = $7,
				start_date = $8::date,
				end_date = nullif($9, '')::date,
				updated_at = now()
			where id::text = $1
			returning id::text
		`, req.ID, req.UserID, req.PositionCode, req.OrgUnitCode, req.OrganizationName, req.IsPrimary, req.Active, req.StartDate, req.EndDate).Scan(&item.ID)
	}
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_membership_save_failed"})
		return
	}

	if _, err := tx.Exec(r.Context(), `
		insert into app_user_modules(user_id, module_code)
		select $1::uuid, scope_module
		from app_positions
		where code = $2 and active = true
		on conflict do nothing
	`, req.UserID, req.PositionCode); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_membership_save_failed"})
		return
	}

	err = tx.QueryRow(r.Context(), `
		select
			m.id::text,
			u.id::text,
			u.name,
			u.email,
			m.position_code,
			p.name,
			m.org_unit_code,
			coalesce(ou.name, m.organization_name),
			m.is_primary,
			m.active,
			to_char(m.start_date, 'YYYY-MM-DD'),
			coalesce(to_char(m.end_date, 'YYYY-MM-DD'), '')
		from app_memberships m
		join app_users u on u.id = m.user_id
		join app_positions p on p.code = m.position_code
		join app_org_units ou on ou.code = m.org_unit_code
		where m.id::text = $1
	`, item.ID).Scan(&item.ID, &item.UserID, &item.UserName, &item.UserEmail, &item.PositionCode, &item.PositionName, &item.OrgUnitCode, &item.OrganizationName, &item.IsPrimary, &item.Active, &item.StartDate, &item.EndDate)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_membership_save_failed"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_membership_save_failed"})
		return
	}

	s.logAudit(r, "admin.memberships.upsert", "membership", item.ID, "Saved membership assignment", map[string]any{
		"user_id":       item.UserID,
		"position_code": item.PositionCode,
		"org_unit_code": item.OrgUnitCode,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) ListPositions(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{"name": {}, "scope_module": {}, "active": {}},
		[]string{"name", "scope_module", "sort_order"},
	)
	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["name"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(name) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["scope_module"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "scope_module = $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["active"]); value != "" {
		args = append(args, value == "true")
		clauses = append(clauses, "active = $"+strconv.Itoa(len(args)))
	}
	whereClause := "where " + strings.Join(clauses, " and ")

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from app_positions "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_positions_failed"})
		return
	}
	sortField := "name"
	switch query.Sort {
	case "scope_module":
		sortField = "scope_module"
	case "sort_order":
		sortField = "sort_order"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select code, name, scope_module, active, sort_order
		from app_positions
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, name
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_positions_failed"})
		return
	}
	defer rows.Close()

	items := []Position{}
	for rows.Next() {
		var item Position
		if err := rows.Scan(&item.Code, &item.Name, &item.ScopeModule, &item.Active, &item.SortOrder); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_positions_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_positions_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) UpsertPosition(w http.ResponseWriter, r *http.Request) {
	var req UpsertPositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_position"})
		return
	}

	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	req.ScopeModule = strings.TrimSpace(req.ScopeModule)
	if req.Code == "" || req.Name == "" || req.ScopeModule == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_position"})
		return
	}

	var item Position
	err := s.pool.QueryRow(r.Context(), `
		insert into app_positions (code, name, scope_module, active, sort_order)
		values ($1, $2, $3, $4, $5)
		on conflict (code) do update
		set name = excluded.name,
			scope_module = excluded.scope_module,
			active = excluded.active,
			sort_order = excluded.sort_order,
			updated_at = now()
		returning code, name, scope_module, active, sort_order
	`, req.Code, req.Name, req.ScopeModule, req.Active, req.SortOrder).Scan(
		&item.Code, &item.Name, &item.ScopeModule, &item.Active, &item.SortOrder,
	)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_position_save_failed"})
		return
	}

	httpx.JSON(w, http.StatusCreated, item)
	s.logAudit(r, "admin.positions.upsert", "position", item.Code, "Saved operational position", map[string]any{
		"scope_module": item.ScopeModule,
		"active":       item.Active,
	})
}

func (s *Service) ListPermissions(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{"code": {}, "label": {}},
		[]string{"code", "label", "user_count"},
	)
	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["code"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(p.code) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["label"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(p.label) like $"+strconv.Itoa(len(args)))
	}
	whereClause := "where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), `select count(*) from app_permissions p `+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permissions_failed"})
		return
	}
	sortField := "p.code"
	switch query.Sort {
	case "label":
		sortField = "p.label"
	case "user_count":
		sortField = "user_count"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			p.code,
			p.label,
			coalesce(up.user_count, 0),
			coalesce(pm.position_count, 0)
		from app_permissions p
		left join (
			select permission_code, count(*) as user_count
			from app_user_permissions
			group by permission_code
		) up on up.permission_code = p.code
		left join (
			select permission_code, count(distinct position_code) as position_count
			from app_position_permissions
			group by permission_code
		) pm on pm.permission_code = p.code
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, p.code
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permissions_failed"})
		return
	}
	defer rows.Close()

	items := []Permission{}
	for rows.Next() {
		var item Permission
		if err := rows.Scan(&item.Code, &item.Label, &item.UserCount, &item.RoleCount); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permissions_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permissions_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PermissionAssignmentFilters(w http.ResponseWriter, r *http.Request) {
	permissionRows, err := s.pool.Query(r.Context(), `
		select code, label
		from app_permissions
		order by code
	`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permission_assignments_filters_failed"})
		return
	}
	defer permissionRows.Close()

	permissions := []map[string]string{}
	for permissionRows.Next() {
		var code, label string
		if err := permissionRows.Scan(&code, &label); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permission_assignments_filters_failed"})
			return
		}
		permissions = append(permissions, map[string]string{"code": code, "label": label})
	}
	if err := permissionRows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permission_assignments_filters_failed"})
		return
	}

	positionRows, err := s.pool.Query(r.Context(), `
		select code, name
		from app_positions
		order by sort_order, name
	`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permission_assignments_filters_failed"})
		return
	}
	defer positionRows.Close()

	positions := []map[string]string{}
	for positionRows.Next() {
		var code, name string
		if err := positionRows.Scan(&code, &name); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permission_assignments_filters_failed"})
			return
		}
		positions = append(positions, map[string]string{"code": code, "name": name})
	}
	if err := positionRows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permission_assignments_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"permissions": permissions,
		"positions":   positions,
	})
}

func (s *Service) ListPermissionAssignments(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{"permission_code": {}, "position_code": {}, "scope_module": {}},
		[]string{"permission_code", "position_name", "scope_module"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["permission_code"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "pap.permission_code = $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["position_code"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "pap.position_code = $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["scope_module"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "p.scope_module = $"+strconv.Itoa(len(args)))
	}
	whereClause := "where " + strings.Join(clauses, " and ")

	var total int
	if err := s.pool.QueryRow(r.Context(), `
		select count(*)
		from app_position_permissions pap
		join app_positions p on p.code = pap.position_code
		`+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permission_assignments_failed"})
		return
	}

	sortField := "pap.permission_code"
	switch query.Sort {
	case "position_name":
		sortField = "p.name"
	case "scope_module":
		sortField = "p.scope_module"
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			pap.permission_code || ':' || pap.position_code as id,
			pap.permission_code,
			ap.label,
			pap.position_code,
			p.name,
			p.scope_module
		from app_position_permissions pap
		join app_permissions ap on ap.code = pap.permission_code
		join app_positions p on p.code = pap.position_code
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, pap.permission_code, p.name
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permission_assignments_failed"})
		return
	}
	defer rows.Close()

	items := []PermissionAssignment{}
	for rows.Next() {
		var item PermissionAssignment
		if err := rows.Scan(&item.ID, &item.PermissionCode, &item.PermissionLabel, &item.PositionCode, &item.PositionName, &item.ScopeModule); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permission_assignments_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_permission_assignments_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) UpsertPermissionAssignment(w http.ResponseWriter, r *http.Request) {
	var req UpsertPermissionAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_permission_assignment"})
		return
	}
	req.PermissionCode = strings.TrimSpace(req.PermissionCode)
	req.PositionCode = strings.TrimSpace(req.PositionCode)
	if req.PermissionCode == "" || req.PositionCode == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_permission_assignment"})
		return
	}

	if req.Assigned {
		if _, err := s.pool.Exec(r.Context(), `
			insert into app_position_permissions(position_code, permission_code)
			values ($1, $2)
			on conflict do nothing
		`, req.PositionCode, req.PermissionCode); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_permission_assignment_save_failed"})
			return
		}
	} else {
		if _, err := s.pool.Exec(r.Context(), `
			delete from app_position_permissions
			where position_code = $1 and permission_code = $2
		`, req.PositionCode, req.PermissionCode); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_permission_assignment_save_failed"})
			return
		}
	}

	var item PermissionAssignment
	err := s.pool.QueryRow(r.Context(), `
		select
			$1 || ':' || $2 as id,
			$1,
			ap.label,
			$2,
			p.name,
			p.scope_module
		from app_permissions ap
		join app_positions p on p.code = $2
		where ap.code = $1
	`, req.PermissionCode, req.PositionCode).Scan(&item.ID, &item.PermissionCode, &item.PermissionLabel, &item.PositionCode, &item.PositionName, &item.ScopeModule)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_permission_assignment_save_failed"})
		return
	}

	s.logAudit(r, "admin.permissions.assignment", "position_permission", item.ID, "Updated position permission assignment", map[string]any{
		"permission_code": item.PermissionCode,
		"position_code":   item.PositionCode,
		"assigned":        req.Assigned,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) ListDossierRequirements(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"source_module":          {},
			"relation_type":          {},
			"required_for_submit":    {},
			"required_for_approve":   {},
			"required_for_readiness": {},
		},
		[]string{"source_module", "relation_type", "min_count"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["source_module"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "source_module = $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["relation_type"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "relation_type = $"+strconv.Itoa(len(args)))
	}
	for _, filter := range []struct {
		key    string
		column string
	}{
		{key: "required_for_submit", column: "required_for_submit"},
		{key: "required_for_approve", column: "required_for_approve"},
		{key: "required_for_readiness", column: "required_for_readiness"},
	} {
		if value := strings.TrimSpace(query.Filters[filter.key]); value != "" {
			args = append(args, value == "true")
			clauses = append(clauses, filter.column+" = $"+strconv.Itoa(len(args)))
		}
	}

	whereClause := "where " + strings.Join(clauses, " and ")

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from workflow_dossier_requirements "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_dossier_requirements_failed"})
		return
	}

	sortField := "source_module"
	switch query.Sort {
	case "relation_type":
		sortField = "relation_type"
	case "min_count":
		sortField = "min_count"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			id::text,
			source_module,
			relation_type,
			min_count,
			required_for_readiness,
			required_for_submit,
			required_for_approve
		from workflow_dossier_requirements
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, source_module, relation_type
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_dossier_requirements_failed"})
		return
	}
	defer rows.Close()

	items := []DossierRequirement{}
	for rows.Next() {
		var item DossierRequirement
		if err := rows.Scan(
			&item.ID,
			&item.SourceModule,
			&item.RelationType,
			&item.MinCount,
			&item.RequiredForReadiness,
			&item.RequiredForSubmit,
			&item.RequiredForApprove,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_dossier_requirements_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_dossier_requirements_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) DossierRequirementFilters(w http.ResponseWriter, r *http.Request) {
	load := func(sql string) ([]string, error) {
		rows, err := s.pool.Query(r.Context(), sql)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		values := []string{}
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, rows.Err()
	}

	sourceModules, err := load("select distinct source_module from workflow_dossier_requirements order by source_module")
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_dossier_requirements_filters_failed"})
		return
	}
	relationTypes, err := load("select distinct relation_type from workflow_dossier_requirements order by relation_type")
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_dossier_requirements_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"source_modules": sourceModules,
		"relation_types": relationTypes,
	})
}

func (s *Service) UpsertDossierRequirement(w http.ResponseWriter, r *http.Request) {
	var req CreateDossierRequirementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_dossier_requirement"})
		return
	}

	req.SourceModule = strings.TrimSpace(req.SourceModule)
	req.RelationType = strings.TrimSpace(req.RelationType)
	if req.SourceModule == "" || req.RelationType == "" || req.MinCount < 1 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_dossier_requirement"})
		return
	}

	var item DossierRequirement
	err := s.pool.QueryRow(r.Context(), `
		insert into workflow_dossier_requirements (
			source_module,
			relation_type,
			min_count,
			required_for_readiness,
			required_for_submit,
			required_for_approve
		) values ($1, $2, $3, $4, $5, $6)
		on conflict (source_module, relation_type) do update
		set min_count = excluded.min_count,
			required_for_readiness = excluded.required_for_readiness,
			required_for_submit = excluded.required_for_submit,
			required_for_approve = excluded.required_for_approve,
			updated_at = now()
		returning
			id::text,
			source_module,
			relation_type,
			min_count,
			required_for_readiness,
			required_for_submit,
			required_for_approve
	`, req.SourceModule, req.RelationType, req.MinCount, req.RequiredForReadiness, req.RequiredForSubmit, req.RequiredForApprove).Scan(
		&item.ID,
		&item.SourceModule,
		&item.RelationType,
		&item.MinCount,
		&item.RequiredForReadiness,
		&item.RequiredForSubmit,
		&item.RequiredForApprove,
	)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_dossier_requirement_save_failed"})
		return
	}

	httpx.JSON(w, http.StatusCreated, item)
	s.logAudit(r, "admin.dossier_requirements.upsert", "dossier_requirement", item.ID, "Saved dossier requirement", map[string]any{
		"source_module": item.SourceModule,
		"relation_type": item.RelationType,
	})
}

func (s *Service) ListEducationTaxonomies(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"domain": {},
			"code":   {},
			"active": {},
		},
		[]string{"domain", "code", "sort_order"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["domain"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "domain = $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["code"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(code) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["active"]); value != "" {
		args = append(args, value == "true")
		clauses = append(clauses, "active = $"+strconv.Itoa(len(args)))
	}

	whereClause := "where " + strings.Join(clauses, " and ")

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_taxonomies "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_education_taxonomies_failed"})
		return
	}

	sortField := "domain"
	switch query.Sort {
	case "code":
		sortField = "code"
	case "sort_order":
		sortField = "sort_order"
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			id::text,
			domain,
			code,
			label_ro,
			label_en,
			active,
			sort_order
		from education_taxonomies
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, domain, sort_order, code
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_education_taxonomies_failed"})
		return
	}
	defer rows.Close()

	items := []EducationTaxonomy{}
	for rows.Next() {
		var item EducationTaxonomy
		if err := rows.Scan(
			&item.ID,
			&item.Domain,
			&item.Code,
			&item.LabelRO,
			&item.LabelEN,
			&item.Active,
			&item.SortOrder,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_education_taxonomies_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_education_taxonomies_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) EducationTaxonomyFilters(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), "select distinct domain from education_taxonomies order by domain")
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_education_taxonomies_filters_failed"})
		return
	}
	defer rows.Close()

	domains := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_education_taxonomies_filters_failed"})
			return
		}
		domains = append(domains, value)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_education_taxonomies_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"domains": domains,
	})
}

func (s *Service) UpsertEducationTaxonomy(w http.ResponseWriter, r *http.Request) {
	var req CreateEducationTaxonomyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_education_taxonomy"})
		return
	}

	req.Domain = strings.TrimSpace(req.Domain)
	req.Code = strings.TrimSpace(req.Code)
	req.LabelRO = strings.TrimSpace(req.LabelRO)
	req.LabelEN = strings.TrimSpace(req.LabelEN)

	if req.Domain == "" || req.Code == "" || req.LabelRO == "" || req.LabelEN == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_education_taxonomy"})
		return
	}

	var item EducationTaxonomy
	err := s.pool.QueryRow(r.Context(), `
		insert into education_taxonomies (
			domain,
			code,
			label_ro,
			label_en,
			active,
			sort_order
		) values ($1, $2, $3, $4, $5, $6)
		on conflict (domain, code) do update
		set label_ro = excluded.label_ro,
			label_en = excluded.label_en,
			active = excluded.active,
			sort_order = excluded.sort_order,
			updated_at = now()
		returning
			id::text,
			domain,
			code,
			label_ro,
			label_en,
			active,
			sort_order
	`, req.Domain, req.Code, req.LabelRO, req.LabelEN, req.Active, req.SortOrder).Scan(
		&item.ID,
		&item.Domain,
		&item.Code,
		&item.LabelRO,
		&item.LabelEN,
		&item.Active,
		&item.SortOrder,
	)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_education_taxonomy_save_failed"})
		return
	}

	httpx.JSON(w, http.StatusCreated, item)
	s.logAudit(r, "admin.education_taxonomies.upsert", "education_taxonomy", item.ID, "Saved education taxonomy", map[string]any{
		"domain": item.Domain,
		"code":   item.Code,
	})
}

func (s *Service) ListWorkflowDefinitions(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"code":     {},
			"name":     {},
			"category": {},
			"active":   {},
		},
		[]string{"code", "name", "category", "sla_hours"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["code"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(code) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["name"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(name) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["category"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "category = $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["active"]); value != "" {
		args = append(args, value == "true")
		clauses = append(clauses, "active = $"+strconv.Itoa(len(args)))
	}

	whereClause := "where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from workflow_definitions "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_workflow_definitions_failed"})
		return
	}

	sortField := "name"
	switch query.Sort {
	case "code":
		sortField = "code"
	case "category":
		sortField = "category"
	case "sla_hours":
		sortField = "sla_hours"
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			code,
			name,
			category,
			initial_step,
			sla_hours,
			active
		from workflow_definitions
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, name
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_workflow_definitions_failed"})
		return
	}
	defer rows.Close()

	items := []WorkflowDefinition{}
	for rows.Next() {
		var item WorkflowDefinition
		if err := rows.Scan(
			&item.Code,
			&item.Name,
			&item.Category,
			&item.InitialStep,
			&item.SLAHours,
			&item.Active,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_workflow_definitions_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_workflow_definitions_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) WorkflowDefinitionFilters(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), "select distinct category from workflow_definitions order by category")
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_workflow_definitions_filters_failed"})
		return
	}
	defer rows.Close()

	categories := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_workflow_definitions_filters_failed"})
			return
		}
		categories = append(categories, value)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_workflow_definitions_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"categories": categories})
}

func (s *Service) UpsertWorkflowDefinition(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkflowDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_workflow_definition"})
		return
	}

	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	req.Category = strings.TrimSpace(req.Category)
	req.InitialStep = strings.TrimSpace(req.InitialStep)
	if req.Code == "" || req.Name == "" || req.Category == "" || req.InitialStep == "" || req.SLAHours < 1 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_workflow_definition"})
		return
	}

	var item WorkflowDefinition
	err := s.pool.QueryRow(r.Context(), `
		insert into workflow_definitions (
			code,
			name,
			category,
			initial_step,
			sla_hours,
			active
		) values ($1, $2, $3, $4, $5, $6)
		on conflict (code) do update
		set name = excluded.name,
			category = excluded.category,
			initial_step = excluded.initial_step,
			sla_hours = excluded.sla_hours,
			active = excluded.active
		returning
			code,
			name,
			category,
			initial_step,
			sla_hours,
			active
	`, req.Code, req.Name, req.Category, req.InitialStep, req.SLAHours, req.Active).Scan(
		&item.Code,
		&item.Name,
		&item.Category,
		&item.InitialStep,
		&item.SLAHours,
		&item.Active,
	)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_workflow_definition_save_failed"})
		return
	}

	httpx.JSON(w, http.StatusCreated, item)
	s.logAudit(r, "admin.workflow_definitions.upsert", "workflow_definition", item.Code, "Saved workflow definition", map[string]any{
		"category": item.Category,
		"active":   item.Active,
	})
}

func (s *Service) ListNomenclatures(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"domain": {},
			"code":   {},
			"active": {},
		},
		[]string{"domain", "code", "sort_order"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["domain"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "domain = $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["code"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(code) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["active"]); value != "" {
		args = append(args, value == "true")
		clauses = append(clauses, "active = $"+strconv.Itoa(len(args)))
	}

	whereClause := "where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from app_nomenclatures "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_nomenclatures_failed"})
		return
	}

	sortField := "domain"
	switch query.Sort {
	case "code":
		sortField = "code"
	case "sort_order":
		sortField = "sort_order"
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			id::text,
			domain,
			code,
			label_ro,
			label_en,
			active,
			sort_order
		from app_nomenclatures
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, domain, sort_order, code
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_nomenclatures_failed"})
		return
	}
	defer rows.Close()

	items := []Nomenclature{}
	for rows.Next() {
		var item Nomenclature
		if err := rows.Scan(
			&item.ID,
			&item.Domain,
			&item.Code,
			&item.LabelRO,
			&item.LabelEN,
			&item.Active,
			&item.SortOrder,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_nomenclatures_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_nomenclatures_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) NomenclatureFilters(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), "select distinct domain from app_nomenclatures order by domain")
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_nomenclatures_filters_failed"})
		return
	}
	defer rows.Close()

	domains := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_nomenclatures_filters_failed"})
			return
		}
		domains = append(domains, value)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_nomenclatures_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"domains": domains})
}

func (s *Service) UpsertNomenclature(w http.ResponseWriter, r *http.Request) {
	var req CreateNomenclatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_nomenclature"})
		return
	}

	req.Domain = strings.TrimSpace(req.Domain)
	req.Code = strings.TrimSpace(req.Code)
	req.LabelRO = strings.TrimSpace(req.LabelRO)
	req.LabelEN = strings.TrimSpace(req.LabelEN)
	if req.Domain == "" || req.Code == "" || req.LabelRO == "" || req.LabelEN == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_nomenclature"})
		return
	}

	var item Nomenclature
	err := s.pool.QueryRow(r.Context(), `
		insert into app_nomenclatures (
			domain,
			code,
			label_ro,
			label_en,
			active,
			sort_order
		) values ($1, $2, $3, $4, $5, $6)
		on conflict (domain, code) do update
		set label_ro = excluded.label_ro,
			label_en = excluded.label_en,
			active = excluded.active,
			sort_order = excluded.sort_order,
			updated_at = now()
		returning
			id::text,
			domain,
			code,
			label_ro,
			label_en,
			active,
			sort_order
	`, req.Domain, req.Code, req.LabelRO, req.LabelEN, req.Active, req.SortOrder).Scan(
		&item.ID,
		&item.Domain,
		&item.Code,
		&item.LabelRO,
		&item.LabelEN,
		&item.Active,
		&item.SortOrder,
	)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_nomenclature_save_failed"})
		return
	}

	httpx.JSON(w, http.StatusCreated, item)
	s.logAudit(r, "admin.nomenclatures.upsert", "nomenclature", item.ID, "Saved application nomenclature", map[string]any{
		"domain": item.Domain,
		"code":   item.Code,
	})
}

func (s *Service) ListAuthMethods(w http.ResponseWriter, r *http.Request) {
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from app_auth_methods").Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_auth_methods_failed"})
		return
	}

	rows, err := s.pool.Query(r.Context(), `
		select code, enabled, primary_method, sort_order
		from app_auth_methods
		order by sort_order, code
	`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_auth_methods_failed"})
		return
	}
	defer rows.Close()

	items := []AuthMethodSetting{}
	for rows.Next() {
		var item AuthMethodSetting
		if err := rows.Scan(&item.Code, &item.Enabled, &item.PrimaryMethod, &item.SortOrder); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_auth_methods_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_auth_methods_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, 1, total)
}

func (s *Service) UpsertAuthMethod(w http.ResponseWriter, r *http.Request) {
	var req UpdateAuthMethodSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_auth_method"})
		return
	}

	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_auth_method"})
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_auth_method_save_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	if req.PrimaryMethod {
		if _, err := tx.Exec(r.Context(), "update app_auth_methods set primary_method = false"); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_auth_method_save_failed"})
			return
		}
	}

	var item AuthMethodSetting
	err = tx.QueryRow(r.Context(), `
		insert into app_auth_methods (code, enabled, primary_method, sort_order)
		values ($1, $2, $3, $4)
		on conflict (code) do update
		set enabled = excluded.enabled,
			primary_method = excluded.primary_method,
			sort_order = excluded.sort_order,
			updated_at = now()
		returning code, enabled, primary_method, sort_order
	`, req.Code, req.Enabled, req.PrimaryMethod, req.SortOrder).Scan(
		&item.Code,
		&item.Enabled,
		&item.PrimaryMethod,
		&item.SortOrder,
	)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_auth_method_save_failed"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_auth_method_save_failed"})
		return
	}

	s.logAudit(r, "admin.auth_methods.upsert", "auth_method", item.Code, "Saved authentication method setting", map[string]any{
		"enabled":        item.Enabled,
		"primary_method": item.PrimaryMethod,
		"sort_order":     item.SortOrder,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) ListModules(w http.ResponseWriter, r *http.Request) {
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from app_modules").Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_modules_failed"})
		return
	}

	rows, err := s.pool.Query(r.Context(), `
		select code, active
		from app_modules
		order by code
	`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_modules_failed"})
		return
	}
	defer rows.Close()

	items := []ModuleSetting{}
	for rows.Next() {
		var item ModuleSetting
		if err := rows.Scan(&item.Code, &item.Active); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_modules_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_modules_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, 1, total)
}

func (s *Service) UpsertModule(w http.ResponseWriter, r *http.Request) {
	var req UpdateModuleSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_module"})
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_module"})
		return
	}

	var item ModuleSetting
	err := s.pool.QueryRow(r.Context(), `
		insert into app_modules (code, active)
		values ($1, $2)
		on conflict (code) do update
		set active = excluded.active
		returning code, active
	`, req.Code, req.Active).Scan(&item.Code, &item.Active)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_module_save_failed"})
		return
	}

	s.logAudit(r, "admin.modules.upsert", "module", item.Code, "Saved module activation state", map[string]any{
		"active": item.Active,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) ListOrgUnits(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"name":        {},
			"parent_name": {},
			"active":      {},
		},
		[]string{"name", "parent_name", "sort_order"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["name"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(ou.name) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["parent_name"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(coalesce(parent.name, '')) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["active"]); value != "" {
		args = append(args, value == "true")
		clauses = append(clauses, "ou.active = $"+strconv.Itoa(len(args)))
	}

	whereClause := "where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), `
		select count(*)
		from app_org_units ou
		left join app_org_units parent on parent.code = ou.parent_code
		`+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_org_units_failed"})
		return
	}

	sortField := "ou.sort_order"
	switch query.Sort {
	case "name":
		sortField = "ou.name"
	case "parent_name":
		sortField = "coalesce(parent.name, '')"
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			ou.code,
			ou.name,
			coalesce(ou.parent_code, ''),
			coalesce(parent.name, ''),
			ou.active,
			ou.sort_order
		from app_org_units ou
		left join app_org_units parent on parent.code = ou.parent_code
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, ou.code
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_org_units_failed"})
		return
	}
	defer rows.Close()

	items := []OrgUnit{}
	for rows.Next() {
		var item OrgUnit
		if err := rows.Scan(&item.Code, &item.Name, &item.ParentCode, &item.ParentName, &item.Active, &item.SortOrder); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_org_units_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_org_units_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) UpsertOrgUnit(w http.ResponseWriter, r *http.Request) {
	var req UpsertOrgUnitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_org_unit"})
		return
	}

	req.Code = strings.TrimSpace(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	req.ParentCode = strings.TrimSpace(req.ParentCode)
	if req.Code == "" || req.Name == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_org_unit"})
		return
	}
	if req.ParentCode == req.Code {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_org_unit"})
		return
	}

	var item OrgUnit
	err := s.pool.QueryRow(r.Context(), `
		insert into app_org_units (code, name, parent_code, active, sort_order)
		values ($1, $2, nullif($3, ''), $4, $5)
		on conflict (code) do update
		set name = excluded.name,
			parent_code = excluded.parent_code,
			active = excluded.active,
			sort_order = excluded.sort_order,
			updated_at = now()
		returning code, name, coalesce(parent_code, ''), active, sort_order
	`, req.Code, req.Name, req.ParentCode, req.Active, req.SortOrder).Scan(
		&item.Code,
		&item.Name,
		&item.ParentCode,
		&item.Active,
		&item.SortOrder,
	)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_org_unit_save_failed"})
		return
	}

	if item.ParentCode != "" {
		_ = s.pool.QueryRow(r.Context(), `select coalesce(name, '') from app_org_units where code = $1`, item.ParentCode).Scan(&item.ParentName)
	}

	s.logAudit(r, "admin.org_units.upsert", "org_unit", item.Code, "Saved organizational unit", map[string]any{
		"parent_code": item.ParentCode,
		"active":      item.Active,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) ListGdprSettings(w http.ResponseWriter, r *http.Request) {
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from gdpr_operational_settings").Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_gdpr_settings_failed"})
		return
	}

	rows, err := s.pool.Query(r.Context(), `
		select code, value_type, coalesce(value_text, ''), coalesce(value_bool, false), coalesce(value_int, 0)
		from gdpr_operational_settings
		order by code
	`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_gdpr_settings_failed"})
		return
	}
	defer rows.Close()

	items := []GdprSetting{}
	for rows.Next() {
		var item GdprSetting
		if err := rows.Scan(&item.Code, &item.ValueType, &item.ValueText, &item.ValueBool, &item.ValueInt); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_gdpr_settings_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_gdpr_settings_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, 1, total)
}

func (s *Service) UpsertGdprSetting(w http.ResponseWriter, r *http.Request) {
	var req UpdateGdprSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_gdpr_setting"})
		return
	}

	req.Code = strings.TrimSpace(req.Code)
	req.ValueType = strings.TrimSpace(req.ValueType)
	req.ValueText = strings.TrimSpace(req.ValueText)
	if req.Code == "" || !slices.Contains([]string{"text", "bool", "int"}, req.ValueType) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_gdpr_setting"})
		return
	}
	if req.ValueType == "int" && req.ValueInt < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_gdpr_setting"})
		return
	}
	if req.ValueType == "text" && req.ValueText == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_gdpr_setting"})
		return
	}

	var item GdprSetting
	err := s.pool.QueryRow(r.Context(), `
		insert into gdpr_operational_settings (code, value_type, value_text, value_bool, value_int)
		values ($1, $2, $3, $4, $5)
		on conflict (code) do update
		set value_type = excluded.value_type,
			value_text = excluded.value_text,
			value_bool = excluded.value_bool,
			value_int = excluded.value_int,
			updated_at = now()
		returning code, value_type, coalesce(value_text, ''), coalesce(value_bool, false), coalesce(value_int, 0)
	`,
		req.Code,
		req.ValueType,
		nullableText(req.ValueType, req.ValueText),
		nullableBool(req.ValueType, req.ValueBool),
		nullableInt(req.ValueType, req.ValueInt),
	).Scan(&item.Code, &item.ValueType, &item.ValueText, &item.ValueBool, &item.ValueInt)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_gdpr_setting_save_failed"})
		return
	}

	s.logAudit(r, "admin.gdpr_settings.upsert", "gdpr_setting", item.Code, "Saved GDPR operational setting", map[string]any{
		"value_type": item.ValueType,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func nullableText(valueType, value string) any {
	if valueType != "text" {
		return nil
	}
	return value
}

func nullableBool(valueType string, value bool) any {
	if valueType != "bool" {
		return nil
	}
	return value
}

func nullableInt(valueType string, value int) any {
	if valueType != "int" {
		return nil
	}
	return value
}

func (s *Service) ListOIDCClients(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"client_id": {},
			"active":    {},
		},
		[]string{"client_id", "client_name", "created_at"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["client_id"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "lower(c.client_id) like $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["active"]); value != "" {
		args = append(args, value == "true")
		clauses = append(clauses, "c.active = $"+strconv.Itoa(len(args)))
	}

	whereClause := "where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from oidc_clients c "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_clients_failed"})
		return
	}

	sortField := "c.client_id"
	switch query.Sort {
	case "client_name":
		sortField = "c.client_name"
	case "created_at":
		sortField = "c.created_at"
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			c.client_id,
			c.client_name,
			c.public_client,
			c.require_pkce,
			c.active,
			coalesce(array_remove(array_agg(cru.redirect_uri order by cru.redirect_uri), null), '{}'),
			coalesce(to_char(c.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		from oidc_clients c
		left join oidc_client_redirect_uris cru on cru.client_id = c.client_id
		`+whereClause+`
		group by c.client_id, c.client_name, c.public_client, c.require_pkce, c.active, c.created_at
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, c.client_id
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_clients_failed"})
		return
	}
	defer rows.Close()

	items := []OIDCClient{}
	for rows.Next() {
		var item OIDCClient
		if err := rows.Scan(
			&item.ClientID,
			&item.ClientName,
			&item.PublicClient,
			&item.RequirePKCE,
			&item.Active,
			&item.RedirectURIs,
			&item.CreatedAt,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_clients_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_clients_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) UpsertOIDCClient(w http.ResponseWriter, r *http.Request) {
	var req UpsertOIDCClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_oidc_client"})
		return
	}

	req.ClientID = strings.TrimSpace(req.ClientID)
	req.ClientName = strings.TrimSpace(req.ClientName)
	redirectURIs := []string{}
	for _, uri := range req.RedirectURIs {
		normalized := strings.TrimSpace(uri)
		if normalized != "" && !slices.Contains(redirectURIs, normalized) {
			redirectURIs = append(redirectURIs, normalized)
		}
	}
	if req.ClientID == "" || req.ClientName == "" || len(redirectURIs) == 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_oidc_client"})
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_client_save_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	if _, err := tx.Exec(r.Context(), `
		insert into oidc_clients (client_id, client_name, public_client, require_pkce, active)
		values ($1, $2, $3, $4, $5)
		on conflict (client_id) do update
		set client_name = excluded.client_name,
			public_client = excluded.public_client,
			require_pkce = excluded.require_pkce,
			active = excluded.active,
			updated_at = now()
	`, req.ClientID, req.ClientName, req.PublicClient, req.RequirePKCE, req.Active); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_oidc_client_save_failed"})
		return
	}

	if _, err := tx.Exec(r.Context(), `delete from oidc_client_redirect_uris where client_id = $1`, req.ClientID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_client_save_failed"})
		return
	}
	for _, redirectURI := range redirectURIs {
		if _, err := tx.Exec(r.Context(), `
			insert into oidc_client_redirect_uris (client_id, redirect_uri)
			values ($1, $2)
		`, req.ClientID, redirectURI); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_oidc_client_save_failed"})
			return
		}
	}

	var item OIDCClient
	if err := tx.QueryRow(r.Context(), `
		select
			c.client_id,
			c.client_name,
			c.public_client,
			c.require_pkce,
			c.active,
			coalesce(array_remove(array_agg(cru.redirect_uri order by cru.redirect_uri), null), '{}'),
			coalesce(to_char(c.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		from oidc_clients c
		left join oidc_client_redirect_uris cru on cru.client_id = c.client_id
		where c.client_id = $1
		group by c.client_id, c.client_name, c.public_client, c.require_pkce, c.active, c.created_at
	`, req.ClientID).Scan(
		&item.ClientID,
		&item.ClientName,
		&item.PublicClient,
		&item.RequirePKCE,
		&item.Active,
		&item.RedirectURIs,
		&item.CreatedAt,
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_client_save_failed"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_client_save_failed"})
		return
	}

	s.logAudit(r, "admin.oidc_clients.upsert", "oidc_client", item.ClientID, "Saved OIDC client", map[string]any{
		"public_client": item.PublicClient,
		"require_pkce":  item.RequirePKCE,
		"active":        item.Active,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) ListOIDCConsents(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"client_id": {},
			"subject":   {},
		},
		[]string{"client_id", "subject", "granted_at"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["client_id"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "c.client_id = $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["subject"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "(lower(oc.subject) like $"+strconv.Itoa(len(args))+" or lower(coalesce(u.email, '')) like $"+strconv.Itoa(len(args))+" or lower(coalesce(u.name, '')) like $"+strconv.Itoa(len(args))+")")
	}

	whereClause := "where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), `
		select count(*)
		from oidc_consents oc
		join oidc_clients c on c.client_id = oc.client_id
		left join app_users u on lower(u.sub) = lower(oc.subject)
		`+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_consents_failed"})
		return
	}

	sortField := "oc.granted_at"
	switch query.Sort {
	case "client_id":
		sortField = "c.client_id"
	case "subject":
		sortField = "oc.subject"
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			oc.id::text,
			c.client_id,
			c.client_name,
			oc.subject,
			coalesce(u.name, ''),
			coalesce(u.email, ''),
			oc.scope,
			coalesce(to_char(oc.granted_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		from oidc_consents oc
		join oidc_clients c on c.client_id = oc.client_id
		left join app_users u on lower(u.sub) = lower(oc.subject)
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, c.client_id
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_consents_failed"})
		return
	}
	defer rows.Close()

	items := []OIDCConsentGrant{}
	for rows.Next() {
		var item OIDCConsentGrant
		if err := rows.Scan(
			&item.ID,
			&item.ClientID,
			&item.ClientName,
			&item.Subject,
			&item.SubjectName,
			&item.SubjectEmail,
			&item.Scope,
			&item.GrantedAt,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_consents_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_consents_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) RevokeOIDCConsent(w http.ResponseWriter, r *http.Request) {
	var req RevokeOIDCGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_oidc_consent"})
		return
	}
	req.ID = strings.TrimSpace(req.ID)
	if req.ID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_oidc_consent"})
		return
	}

	command, err := s.pool.Exec(r.Context(), `delete from oidc_consents where id::text = $1`, req.ID)
	if err != nil || command.RowsAffected() == 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_oidc_consent_revoke_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"status": "revoked"})
	s.logAudit(r, "admin.oidc_consents.revoke", "oidc_consent", req.ID, "Revoked OIDC consent grant", nil)
}

func (s *Service) ListOIDCSessions(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"client_id": {},
			"subject":   {},
			"revoked":   {},
		},
		[]string{"client_id", "subject", "created_at", "expires_at"},
	)

	clauses := []string{"1 = 1"}
	args := []any{}
	if value := strings.TrimSpace(query.Filters["client_id"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, "rt.client_id = $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(query.Filters["subject"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, "(lower(rt.subject) like $"+strconv.Itoa(len(args))+" or lower(coalesce(u.email, '')) like $"+strconv.Itoa(len(args))+" or lower(coalesce(u.name, '')) like $"+strconv.Itoa(len(args))+")")
	}
	if value := strings.TrimSpace(query.Filters["revoked"]); value != "" {
		args = append(args, value == "true")
		clauses = append(clauses, "rt.revoked = $"+strconv.Itoa(len(args)))
	}

	whereClause := "where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), `
		select count(*)
		from oidc_refresh_tokens rt
		join oidc_clients c on c.client_id = rt.client_id
		left join app_users u on lower(u.sub) = lower(rt.subject)
		`+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_sessions_failed"})
		return
	}

	sortField := "rt.created_at"
	switch query.Sort {
	case "client_id":
		sortField = "rt.client_id"
	case "subject":
		sortField = "rt.subject"
	case "expires_at":
		sortField = "rt.expires_at"
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select
			rt.token,
			rt.client_id,
			c.client_name,
			rt.subject,
			coalesce(u.name, ''),
			coalesce(u.email, ''),
			rt.scope,
			coalesce(to_char(rt.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
			coalesce(to_char(rt.expires_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
			rt.revoked
		from oidc_refresh_tokens rt
		join oidc_clients c on c.client_id = rt.client_id
		left join app_users u on lower(u.sub) = lower(rt.subject)
		`+whereClause+`
		order by `+sortField+` `+strings.ToUpper(query.Direction)+`, rt.created_at desc
		limit $`+strconv.Itoa(len(args)-1)+` offset $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_sessions_failed"})
		return
	}
	defer rows.Close()

	items := []OIDCSession{}
	for rows.Next() {
		var item OIDCSession
		if err := rows.Scan(
			&item.TokenID,
			&item.ClientID,
			&item.ClientName,
			&item.Subject,
			&item.SubjectName,
			&item.SubjectEmail,
			&item.Scope,
			&item.CreatedAt,
			&item.ExpiresAt,
			&item.Revoked,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_sessions_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admin_oidc_sessions_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) RevokeOIDCSession(w http.ResponseWriter, r *http.Request) {
	var req RevokeOIDCSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_oidc_session"})
		return
	}
	req.TokenID = strings.TrimSpace(req.TokenID)
	if req.TokenID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admin_oidc_session"})
		return
	}

	command, err := s.pool.Exec(r.Context(), `
		update oidc_refresh_tokens
		set revoked = true, updated_at = now()
		where token = $1
			and revoked = false
	`, req.TokenID)
	if err != nil || command.RowsAffected() == 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "admin_oidc_session_revoke_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"status": "revoked"})
	s.logAudit(r, "admin.oidc_sessions.revoke", "oidc_session", req.TokenID, "Revoked OIDC refresh-token session", nil)
}
