package registratura

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type departmentRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	ParentID    *string `json:"parent_id"`
	RoleTag     string  `json:"role_tag"`
	Active      *bool   `json:"active"`
}
type organizationRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Active        *bool    `json:"active"`
	IsDefault     *bool    `json:"is_default"`
	DepartmentIDs []string `json:"department_ids"`
}
type assignmentRequest struct {
	DepartmentIDs       []string `json:"department_ids"`
	PrimaryDepartmentID *string  `json:"primary_department_id"`
	OrganizationID      *string  `json:"organization_id"`
}

func (s *Service) ListDepartments(w http.ResponseWriter, r *http.Request) {
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"query": {}}, []string{"name"})
	filter := strings.TrimSpace(q.Filters["query"])
	args := []any{}
	where := "where 1=1"
	if filter != "" {
		args = append(args, "%"+strings.ToLower(filter)+"%")
		where += " and lower(d.name) like $1"
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from registratura_departments d "+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "departments_list_failed"})
		return
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := s.pool.Query(r.Context(), "select d.id::text,d.name,d.description,d.parent_id::text,d.role_tag,d.active,(select count(*) from registratura_user_departments ud where ud.department_id=d.id) from registratura_departments d "+where+" order by d.name limit $"+itoa(len(args)-1)+" offset $"+itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "departments_list_failed"})
		return
	}
	defer rows.Close()
	items := []Department{}
	for rows.Next() {
		var i Department
		var parent *string
		if err := rows.Scan(&i.ID, &i.Name, &i.Description, &parent, &i.RoleTag, &i.Active, &i.UserCount); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "departments_list_failed"})
			return
		}
		i.ParentID = parent
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "departments_list_failed"})
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}
func (s *Service) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	var req departmentRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.Name) == "" {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_department"})
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	var i Department
	err := s.pool.QueryRow(r.Context(), `insert into registratura_departments(tenant_code,institution_id,name,description,parent_id,role_tag,active) values(public.current_tenant_code(),public.current_institution_id(),$1,$2,$3::uuid,$4,$5) returning id::text,name,description,parent_id::text,role_tag,active`, strings.TrimSpace(req.Name), strings.TrimSpace(req.Description), req.ParentID, strings.TrimSpace(req.RoleTag), active).Scan(&i.ID, &i.Name, &i.Description, &i.ParentID, &i.RoleTag, &i.Active)
	if err != nil {
		httpx.JSON(w, 400, map[string]any{"code": "department_create_failed"})
		return
	}
	httpx.JSON(w, 201, i)
}
func (s *Service) UpdateDepartment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req departmentRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.Name) == "" {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_department"})
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	var i Department
	err := s.pool.QueryRow(r.Context(), `update registratura_departments set name=$2,description=$3,parent_id=$4::uuid,role_tag=$5,active=$6,updated_at=now() where id=$1::uuid returning id::text,name,description,parent_id::text,role_tag,active`, id, strings.TrimSpace(req.Name), strings.TrimSpace(req.Description), req.ParentID, strings.TrimSpace(req.RoleTag), active).Scan(&i.ID, &i.Name, &i.Description, &i.ParentID, &i.RoleTag, &i.Active)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, 404, map[string]any{"code": "department_not_found"})
		} else {
			httpx.JSON(w, 400, map[string]any{"code": "department_update_failed"})
		}
		return
	}
	httpx.JSON(w, 200, i)
}
func (s *Service) DeleteDepartment(w http.ResponseWriter, r *http.Request) {
	tag, err := s.pool.Exec(r.Context(), `delete from registratura_departments where id=$1::uuid and not exists(select 1 from registratura_document_departments where department_id=$1::uuid)`, chi.URLParam(r, "id"))
	if err != nil {
		httpx.JSON(w, 409, map[string]any{"code": "department_delete_blocked"})
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.JSON(w, 404, map[string]any{"code": "department_not_found_or_referenced"})
		return
	}
	w.WriteHeader(204)
}

func (s *Service) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{}, []string{"name"})
	var total int
	if err := s.pool.QueryRow(r.Context(), `select count(*) from registratura_organizations`).Scan(&total); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "organizations_list_failed"})
		return
	}
	rows, err := s.pool.Query(r.Context(), `select o.id::text,o.name,o.description,o.active,o.is_default,coalesce(array_agg(od.department_id::text) filter(where od.department_id is not null),'{}') from registratura_organizations o left join registratura_organization_departments od on od.organization_id=o.id group by o.id order by o.is_default desc,o.name limit $1 offset $2`, q.PageSize, (q.Page-1)*q.PageSize)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "organizations_list_failed"})
		return
	}
	defer rows.Close()
	items := []Organization{}
	for rows.Next() {
		var i Organization
		if err := rows.Scan(&i.ID, &i.Name, &i.Description, &i.Active, &i.IsDefault, &i.DepartmentIDs); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "organizations_list_failed"})
			return
		}
		items = append(items, i)
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}
func (s *Service) saveOrganization(w http.ResponseWriter, r *http.Request, id *string) {
	var req organizationRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.Name) == "" {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_organization"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "organization_save_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	def := false
	if req.IsDefault != nil {
		def = *req.IsDefault
	}
	if def {
		_, err = tx.Exec(r.Context(), `update registratura_organizations set is_default=false where is_default`)
		if err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "organization_save_failed"})
			return
		}
	}
	var oid string
	if id == nil {
		err = tx.QueryRow(r.Context(), `insert into registratura_organizations(tenant_code,institution_id,name,description,active,is_default)values(public.current_tenant_code(),public.current_institution_id(),$1,$2,$3,$4)returning id::text`, strings.TrimSpace(req.Name), strings.TrimSpace(req.Description), active, def).Scan(&oid)
	} else {
		oid = *id
		_, err = tx.Exec(r.Context(), `update registratura_organizations set name=$2,description=$3,active=$4,is_default=$5,updated_at=now()where id=$1::uuid`, oid, strings.TrimSpace(req.Name), strings.TrimSpace(req.Description), active, def)
	}
	if err != nil {
		httpx.JSON(w, 400, map[string]any{"code": "organization_save_failed"})
		return
	}
	if _, err = tx.Exec(r.Context(), `delete from registratura_organization_departments where organization_id=$1::uuid`, oid); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "organization_save_failed"})
		return
	}
	for _, did := range req.DepartmentIDs {
		if _, err = tx.Exec(r.Context(), `insert into registratura_organization_departments(tenant_code,institution_id,organization_id,department_id)values(public.current_tenant_code(),public.current_institution_id(),$1::uuid,$2::uuid)`, oid, did); err != nil {
			httpx.JSON(w, 400, map[string]any{"code": "invalid_organization_department"})
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "organization_save_failed"})
		return
	}
	httpx.JSON(w, map[bool]int{id == nil: 201, id != nil: 200}[true], map[string]any{"id": oid})
}
func (s *Service) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	s.saveOrganization(w, r, nil)
}
func (s *Service) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s.saveOrganization(w, r, &id)
}
func (s *Service) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	tag, err := s.pool.Exec(r.Context(), `delete from registratura_organizations where id=$1::uuid`, chi.URLParam(r, "id"))
	if err != nil {
		httpx.JSON(w, 409, map[string]any{"code": "organization_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.JSON(w, 404, map[string]any{"code": "organization_not_found"})
		return
	}
	w.WriteHeader(204)
}

func itoa(v int) string { return fmt.Sprintf("%d", v) }

type chartNode struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	ParentID    *string      `json:"parent_id,omitempty"`
	RoleTag     string       `json:"role_tag"`
	UserCount   int          `json:"user_count"`
	Users       []chartUser  `json:"users"`
	Children    []*chartNode `json:"children"`
}
type chartUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

func (s *Service) WorkflowAssignees(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		select u.id::text, u.name, coalesce(array_agg(ud.department_id::text) filter(where ud.department_id is not null), '{}')
		from app_memberships m
		join app_users u on u.id=m.user_id
		left join registratura_user_departments ud on ud.user_id=u.id
		where m.active
		group by u.id,u.name order by u.name`)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "workflow_assignees_failed"})
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, name string
		var departments []string
		if err := rows.Scan(&id, &name, &departments); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "workflow_assignees_failed"})
			return
		}
		items = append(items, map[string]any{"id": id, "name": name, "department_ids": departments})
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "workflow_assignees_failed"})
		return
	}
	departmentRows, err := s.pool.Query(r.Context(), `select id::text,name from registratura_departments where active order by name`)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "workflow_assignees_failed"})
		return
	}
	defer departmentRows.Close()
	departments := []map[string]string{}
	for departmentRows.Next() {
		var id, name string
		if err := departmentRows.Scan(&id, &name); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "workflow_assignees_failed"})
			return
		}
		departments = append(departments, map[string]string{"id": id, "name": name})
	}
	httpx.JSON(w, 200, map[string]any{"users": items, "departments": departments})
}

func (s *Service) OrganizationChart(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `select d.id::text,d.name,d.description,d.parent_id::text,d.role_tag,u.id::text,coalesce(u.name,''),coalesce(u.email,'') from registratura_departments d left join registratura_user_departments ud on ud.department_id=d.id left join app_users u on u.id=ud.user_id where d.active order by d.name,u.name`)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "organization_chart_failed"})
		return
	}
	defer rows.Close()
	nodes := map[string]*chartNode{}
	order := []string{}
	for rows.Next() {
		var id, name, desc, role string
		var parent, userID, userName, userEmail *string
		if err := rows.Scan(&id, &name, &desc, &parent, &role, &userID, &userName, &userEmail); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "organization_chart_failed"})
			return
		}
		n := nodes[id]
		if n == nil {
			n = &chartNode{ID: id, Name: name, Description: desc, ParentID: parent, RoleTag: role, Users: []chartUser{}, Children: []*chartNode{}}
			nodes[id] = n
			order = append(order, id)
		}
		if userID != nil {
			n.Users = append(n.Users, chartUser{ID: *userID, Name: *userName, Email: *userEmail})
		}
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "organization_chart_failed"})
		return
	}
	roots := []*chartNode{}
	for _, id := range order {
		n := nodes[id]
		n.UserCount = len(n.Users)
		if n.ParentID != nil && nodes[*n.ParentID] != nil {
			nodes[*n.ParentID].Children = append(nodes[*n.ParentID].Children, n)
		} else {
			roots = append(roots, n)
		}
	}
	httpx.JSON(w, 200, roots)
}

func (s *Service) GetUserAssignments(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	rows, err := s.pool.Query(r.Context(), `select department_id::text,is_primary from registratura_user_departments where user_id=$1::uuid order by is_primary desc`, userID)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "user_assignments_failed"})
		return
	}
	defer rows.Close()
	result := map[string]any{"user_id": userID, "department_ids": []string{}, "primary_department_id": nil, "organization_id": nil}
	ids := []string{}
	var primary *string
	for rows.Next() {
		var id string
		var isPrimary bool
		if err := rows.Scan(&id, &isPrimary); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "user_assignments_failed"})
			return
		}
		ids = append(ids, id)
		if isPrimary {
			primary = &id
		}
	}
	result["department_ids"] = ids
	result["primary_department_id"] = primary
	var org sql.NullString
	if err := s.pool.QueryRow(r.Context(), `select organization_id::text from registratura_user_organizations where user_id=$1::uuid`, userID).Scan(&org); err == nil {
		result["organization_id"] = org.String
	} else if err != pgx.ErrNoRows {
		httpx.JSON(w, 500, map[string]any{"code": "user_assignments_failed"})
		return
	}
	httpx.JSON(w, 200, result)
}
func (s *Service) PutUserAssignments(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	var req assignmentRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_user_assignments"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "user_assignments_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	var member bool
	if err = tx.QueryRow(r.Context(), `select exists(select 1 from app_memberships where user_id=$1::uuid and active)`, userID).Scan(&member); err != nil || !member {
		httpx.JSON(w, 400, map[string]any{"code": "user_not_in_tenant"})
		return
	}
	if req.PrimaryDepartmentID != nil && !contains(req.DepartmentIDs, *req.PrimaryDepartmentID) {
		httpx.JSON(w, 400, map[string]any{"code": "primary_department_not_assigned"})
		return
	}
	if _, err = tx.Exec(r.Context(), `delete from registratura_user_departments where user_id=$1::uuid`, userID); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "user_assignments_failed"})
		return
	}
	for _, dep := range req.DepartmentIDs {
		primary := req.PrimaryDepartmentID != nil && dep == *req.PrimaryDepartmentID
		if _, err = tx.Exec(r.Context(), `insert into registratura_user_departments(tenant_code,institution_id,user_id,department_id,is_primary)values(public.current_tenant_code(),public.current_institution_id(),$1::uuid,$2::uuid,$3)`, userID, dep, primary); err != nil {
			httpx.JSON(w, 400, map[string]any{"code": "invalid_user_department"})
			return
		}
	}
	if _, err = tx.Exec(r.Context(), `delete from registratura_user_organizations where user_id=$1::uuid`, userID); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "user_assignments_failed"})
		return
	}
	if req.OrganizationID != nil && strings.TrimSpace(*req.OrganizationID) != "" {
		if _, err = tx.Exec(r.Context(), `insert into registratura_user_organizations(tenant_code,institution_id,user_id,organization_id)values(public.current_tenant_code(),public.current_institution_id(),$1::uuid,$2::uuid)`, userID, *req.OrganizationID); err != nil {
			httpx.JSON(w, 400, map[string]any{"code": "invalid_user_organization"})
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "user_assignments_failed"})
		return
	}
	s.GetUserAssignments(w, r)
}

type adminRegistry struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Prefix        string   `json:"prefix"`
	StartNumber   int      `json:"start_number"`
	CurrentNumber string   `json:"current_number"`
	NextNumber    string   `json:"next_number"`
	RegistryType  string   `json:"registry_type"`
	IsDefault     bool     `json:"is_default"`
	DepartmentIDs []string `json:"department_ids"`
}
type adminRegistryRequest struct {
	Name          string   `json:"name"`
	Prefix        string   `json:"prefix"`
	StartNumber   int      `json:"start_number"`
	CurrentNumber string   `json:"current_number"`
	NextNumber    string   `json:"next_number"`
	RegistryType  string   `json:"registry_type"`
	IsDefault     bool     `json:"is_default"`
	DepartmentIDs []string `json:"department_ids"`
}

func (s *Service) ListAdminRegistries(w http.ResponseWriter, r *http.Request) {
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"query": {}}, []string{"name"})
	where := "where active"
	args := []any{}
	if v := strings.TrimSpace(q.Filters["query"]); v != "" {
		args = append(args, "%"+strings.ToLower(v)+"%")
		where += " and lower(nume) like $1"
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from registre "+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "registries_list_failed"})
		return
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := s.pool.Query(r.Context(), `select r.id,r.nume,r.prefix_nr,r.nr_inceput,r.nr_curent,r.nr_urmator,r.visibility,r.is_default,coalesce(array_agg(rd.department_id::text)filter(where rd.department_id is not null),'{}') from registre r left join registratura_registry_departments rd on rd.registry_id=r.id `+where+` group by r.id order by r.is_default desc,r.nume limit $`+itoa(len(args)-1)+` offset $`+itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "registries_list_failed"})
		return
	}
	defer rows.Close()
	items := []adminRegistry{}
	for rows.Next() {
		var i adminRegistry
		if err := rows.Scan(&i.ID, &i.Name, &i.Prefix, &i.StartNumber, &i.CurrentNumber, &i.NextNumber, &i.RegistryType, &i.IsDefault, &i.DepartmentIDs); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "registries_list_failed"})
			return
		}
		items = append(items, i)
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}
func (s *Service) SaveAdminRegistry(w http.ResponseWriter, r *http.Request) {
	var req adminRegistryRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Prefix) == "" {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_registry"})
		return
	}
	if req.RegistryType != "public" && req.RegistryType != "private" {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_registry_type"})
		return
	}
	if req.StartNumber < 1 {
		req.StartNumber = 1
	}
	if req.CurrentNumber == "" {
		req.CurrentNumber = fmt.Sprintf("%04d", req.StartNumber-1)
	}
	if req.NextNumber == "" {
		req.NextNumber = fmt.Sprintf("%04d", req.StartNumber)
	}
	idRaw := chi.URLParam(r, "id")
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "registry_save_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	if req.IsDefault {
		if _, err = tx.Exec(r.Context(), `update registre set is_default=false where is_default`); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "registry_save_failed"})
			return
		}
	}
	var id int64
	if idRaw == "" {
		err = tx.QueryRow(r.Context(), `insert into registre(tenant_code,institution_id,nume,prefix_nr,nr_inceput,nr_curent,nr_urmator,visibility,is_default) values(public.current_tenant_code(),public.current_institution_id(),$1,$2,$3,$4,$5,$6,$7)returning id`, strings.TrimSpace(req.Name), strings.TrimSpace(req.Prefix), req.StartNumber, req.CurrentNumber, req.NextNumber, req.RegistryType, req.IsDefault).Scan(&id)
	} else {
		var parseErr error
		id, parseErr = strconv.ParseInt(idRaw, 10, 64)
		if parseErr != nil {
			httpx.JSON(w, 400, map[string]any{"code": "invalid_registry_id"})
			return
		}
		_, err = tx.Exec(r.Context(), `update registre set nume=$2,prefix_nr=$3,nr_inceput=$4,nr_curent=$5,nr_urmator=$6,visibility=$7,is_default=$8,updated_at=now() where id=$1`, id, strings.TrimSpace(req.Name), strings.TrimSpace(req.Prefix), req.StartNumber, req.CurrentNumber, req.NextNumber, req.RegistryType, req.IsDefault)
	}
	if err != nil {
		httpx.JSON(w, 400, map[string]any{"code": "registry_save_failed"})
		return
	}
	if _, err = tx.Exec(r.Context(), `delete from registratura_registry_departments where registry_id=$1`, id); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "registry_save_failed"})
		return
	}
	for _, dep := range req.DepartmentIDs {
		if _, err = tx.Exec(r.Context(), `insert into registratura_registry_departments(tenant_code,institution_id,registry_id,department_id)values(public.current_tenant_code(),public.current_institution_id(),$1,$2::uuid)`, id, dep); err != nil {
			httpx.JSON(w, 400, map[string]any{"code": "invalid_registry_department"})
			return
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "registry_save_failed"})
		return
	}
	httpx.JSON(w, 200, map[string]any{"id": id})
}
