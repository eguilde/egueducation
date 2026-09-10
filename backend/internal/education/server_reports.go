package education

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

type SchoolReportColumn struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type SchoolReportCatalogItem struct {
	Code        string               `json:"code"`
	Label       string               `json:"label"`
	Description string               `json:"description"`
	Columns     []SchoolReportColumn `json:"columns"`
	Formats     []string             `json:"formats"`
}

type schoolReportSpec struct {
	catalog         SchoolReportCatalogItem
	baseSQL         string
	defaultSort     string
	readPermissions []string
}

var schoolReportSpecs = map[string]schoolReportSpec{
	"portfolio-status": {
		catalog: SchoolReportCatalogItem{Code: "portfolio-status", Label: "Situația portofoliilor", Description: "Starea portofoliilor profesionale pe titular și an școlar.", Columns: schoolReportColumns("portfolio_code", "Cod", "owner_name", "Titular", "owner_role", "Funcție", "school_year", "An școlar", "status", "Stare", "transfer_status", "Transfer", "last_updated_on", "Actualizat la"), Formats: []string{"json", "csv", "pdf"}},
		baseSQL: `select portfolio_code, owner_name, owner_role, school_year, status, transfer_status, last_updated_on
			from education_portfolios where institution_id=$1`,
		defaultSort:     "last_updated_on",
		readPermissions: []string{"education.portfolios.school.read", "education.portfolios.read"},
	},
	"evaluation-status": {
		catalog: SchoolReportCatalogItem{Code: "evaluation-status", Label: "Situația evaluărilor", Description: "Evaluări profesionale și rezultate în instituția curentă.", Columns: schoolReportColumns("evaluation_code", "Cod", "employee_code", "Angajat", "full_name", "Nume", "school_year", "An școlar", "status", "Stare", "score", "Punctaj", "qualification", "Calificativ", "finalized_on", "Finalizat la"), Formats: []string{"json", "csv", "pdf"}},
		baseSQL: `select evaluation_code, employee_code, full_name, school_year, status, score, qualification, finalized_on
			from education_evaluations where institution_id=$1`,
		defaultSort:     "full_name",
		readPermissions: []string{"education.evaluations.read"},
	},
	"governance-compliance": {
		catalog: SchoolReportCatalogItem{Code: "governance-compliance", Label: "Conformitate ședințe", Description: "Ședințe, minute și voturi păstrate în dosarul de guvernanță.", Columns: schoolReportColumns("title", "Ședință", "organism", "Organism", "meeting_date", "Data", "status", "Stare", "has_minute", "Are minută", "has_vote", "Are vot"), Formats: []string{"json", "csv", "pdf"}},
		baseSQL: `select meeting.title, meeting.organism, meeting.meeting_date, meeting.status,
			exists(select 1 from education_meeting_minutes minute where minute.meeting_id=meeting.id and minute.institution_id=meeting.institution_id) as has_minute,
			exists(select 1 from education_meeting_votes vote where vote.meeting_id=meeting.id and vote.institution_id=meeting.institution_id) as has_vote
			from education_meetings meeting where meeting.institution_id=$1`,
		defaultSort:     "meeting_date",
		readPermissions: []string{"education.governance.read"},
	},
	"personnel-document-expiry": {
		catalog: SchoolReportCatalogItem{Code: "personnel-document-expiry", Label: "Documente de personal", Description: "Documente existente, expirate sau apropiate de expirare din dosarul de personal.", Columns: schoolReportColumns("employee_code", "Angajat", "full_name", "Nume", "document_type", "Tip document", "title", "Document", "status", "Stare", "expires_on", "Expiră la"), Formats: []string{"json", "csv", "pdf"}},
		baseSQL: `select person.employee_code, person.full_name, document.document_category as document_type,
			document.document_title as title,
			case
				when document.expires_on is null then 'fara_expirare'
				when document.expires_on < current_date then 'expirat'
				when document.expires_on <= current_date + 30 then 'expira_curand'
				else 'valabil'
			end as status,
			document.expires_on
			from education_personnel_file_documents document
			join education_personnel person on person.id=document.personnel_id and person.institution_id=document.institution_id
			where document.institution_id=$1`,
		defaultSort:     "expires_on",
		readPermissions: []string{"education.personnel.files.read"},
	},
	"publication-backlog": {
		catalog: SchoolReportCatalogItem{Code: "publication-backlog", Label: "Publicări și anonimizare", Description: "Publicări obligatorii și starea verificării de anonimizare.", Columns: schoolReportColumns("publication_code", "Cod", "entity_type", "Tip", "entity_label", "Entitate", "publication_channel", "Canal", "publication_status", "Stare", "anonymization_status", "Anonimizare", "mandatory", "Obligatorie", "published_on", "Publicat la", "reviewed_by", "Revizuit de"), Formats: []string{"json", "csv", "pdf"}},
		baseSQL: `select publication_code, entity_type, entity_label, publication_channel, publication_status, anonymization_status, mandatory, published_on, reviewed_by
			from education_publications where institution_id=$1`,
		defaultSort:     "publication_code",
		readPermissions: []string{"education.compliance.read"},
	},
}

func schoolReportColumns(values ...string) []SchoolReportColumn {
	columns := make([]SchoolReportColumn, 0, len(values)/2)
	for index := 0; index+1 < len(values); index += 2 {
		columns = append(columns, SchoolReportColumn{Key: values[index], Label: values[index+1]})
	}
	return columns
}

func (s *Service) SchoolReportCatalog(w http.ResponseWriter, r *http.Request) {
	order := []string{"portfolio-status", "evaluation-status", "governance-compliance", "personnel-document-expiry", "publication-backlog"}
	items := make([]SchoolReportCatalogItem, 0, len(order))
	for _, code := range order {
		spec := schoolReportSpecs[code]
		allowed, err := s.canReadSchoolReport(r, spec)
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_report_authorization_failed"})
			return
		}
		if allowed {
			items = append(items, spec.catalog)
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Service) SchoolReport(w http.ResponseWriter, r *http.Request) {
	spec, ok := schoolReportSpecs[chi.URLParam(r, "reportCode")]
	if !ok {
		writeEducationNotFound(w, "education_report_not_found")
		return
	}
	if !s.requireSchoolReportRead(w, r, spec) {
		return
	}
	allowed := make(map[string]struct{}, len(spec.catalog.Columns))
	for _, column := range spec.catalog.Columns {
		allowed[column.Key] = struct{}{}
	}
	query := httpx.ParsePageQuery(r.URL.Query(), allowed, reportColumnKeys(spec.catalog.Columns))
	if query.Sort == "" {
		query.Sort = spec.defaultSort
	}
	rows, total, err := s.loadSchoolReport(r, spec, query.Filters, query.Sort, query.Direction, query.PageSize, (query.Page-1)*query.PageSize)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_report_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, rows, total, query.Page, query.PageSize)
}

func (s *Service) SchoolReportCSV(w http.ResponseWriter, r *http.Request) {
	s.schoolReportDownload(w, r, "csv")
}

func (s *Service) SchoolReportPDF(w http.ResponseWriter, r *http.Request) {
	s.schoolReportDownload(w, r, "pdf")
}

func (s *Service) schoolReportDownload(w http.ResponseWriter, r *http.Request, format string) {
	if !s.requireSchoolReportExport(w, r) {
		return
	}
	code := chi.URLParam(r, "reportCode")
	spec, ok := schoolReportSpecs[code]
	if !ok {
		writeEducationNotFound(w, "education_report_not_found")
		return
	}
	if !s.requireSchoolReportRead(w, r, spec) {
		return
	}
	filters := make(map[string]string)
	for _, column := range spec.catalog.Columns {
		filters[column.Key] = strings.TrimSpace(r.URL.Query().Get("filter." + column.Key))
	}
	rows, _, err := s.loadSchoolReport(r, spec, filters, spec.defaultSort, "asc", 5000, 0)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_report_export_failed"})
		return
	}
	if err := audit.Log(r.Context(), s.pool, audit.Event{
		ActorSubject: authruntime.CurrentSubjectFromRequest(r),
		Action:       "education.reports.export." + format,
		TargetType:   "school_report",
		TargetID:     code,
		Summary:      "School report generated from server-side tenant data.",
		Details:      map[string]any{"row_count": len(rows), "format": format},
	}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_report_audit_failed"})
		return
	}
	if format == "csv" {
		var buffer bytes.Buffer
		buffer.WriteString("\uFEFF")
		writer := csv.NewWriter(&buffer)
		headings := make([]string, 0, len(spec.catalog.Columns))
		for _, column := range spec.catalog.Columns {
			headings = append(headings, column.Label)
		}
		_ = writer.Write(headings)
		for _, row := range rows {
			_ = writer.Write(reportRowValues(row, spec.catalog.Columns))
		}
		writer.Flush()
		if writer.Error() != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_report_export_failed"})
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, code))
		_, _ = w.Write(buffer.Bytes())
		return
	}
	lines := []string{strings.Join(reportColumnLabels(spec.catalog.Columns), " | ")}
	for _, row := range rows {
		lines = append(lines, strings.Join(reportRowValues(row, spec.catalog.Columns), " | "))
	}
	writeEducationPDFDownload(w, spec.catalog.Label, code, lines)
}

func (s *Service) requireSchoolReportExport(w http.ResponseWriter, r *http.Request) bool {
	allowed, err := s.authorizeEducationPermission(r, EducationDelegationScope{PermissionCode: "education.reports.export_sensitive", ResourceType: "institution"})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_report_authorization_failed"})
		return false
	}
	if !allowed {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_report_export_permission_required"})
		return false
	}
	return true
}

func (s *Service) canReadSchoolReport(r *http.Request, spec schoolReportSpec) (bool, error) {
	for _, permission := range spec.readPermissions {
		allowed, err := s.authorizeEducationPermission(r, EducationDelegationScope{PermissionCode: permission, ResourceType: "institution"})
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) requireSchoolReportRead(w http.ResponseWriter, r *http.Request, spec schoolReportSpec) bool {
	allowed, err := s.canReadSchoolReport(r, spec)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_report_authorization_failed"})
		return false
	}
	if !allowed {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_report_permission_required"})
		return false
	}
	return true
}

func (s *Service) loadSchoolReport(r *http.Request, spec schoolReportSpec, filters map[string]string, sortField, direction string, limit, offset int) ([]map[string]any, int, error) {
	allowed := make(map[string]struct{}, len(spec.catalog.Columns))
	for _, column := range spec.catalog.Columns {
		allowed[column.Key] = struct{}{}
	}
	if _, ok := allowed[sortField]; !ok {
		sortField = spec.defaultSort
	}
	direction = strings.ToLower(direction)
	if direction != "desc" {
		direction = "asc"
	}
	where := []string{"1=1"}
	args := []any{s.institutionID(r)}
	for _, column := range spec.catalog.Columns {
		if value := strings.TrimSpace(filters[column.Key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(coalesce(report.%s::text, '')) like $%d", column.Key, len(args)))
		}
	}
	whereSQL := strings.Join(where, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from ("+spec.baseSQL+") report where "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	querySQL := "select * from (" + spec.baseSQL + ") report where " + whereSQL + " order by report." + sortField + " " + direction + " nulls last limit $" + strconv.Itoa(len(args)-1) + " offset $" + strconv.Itoa(len(args))
	result, err := s.pool.Query(r.Context(), querySQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer result.Close()
	rows := make([]map[string]any, 0, limit)
	fields := result.FieldDescriptions()
	for result.Next() {
		values, err := result.Values()
		if err != nil {
			return nil, 0, err
		}
		row := make(map[string]any, len(values))
		for index, value := range values {
			row[string(fields[index].Name)] = value
		}
		rows = append(rows, row)
	}
	return rows, total, result.Err()
}

func reportColumnKeys(columns []SchoolReportColumn) []string {
	keys := make([]string, 0, len(columns))
	for _, column := range columns {
		keys = append(keys, column.Key)
	}
	return keys
}

func reportColumnLabels(columns []SchoolReportColumn) []string {
	labels := make([]string, 0, len(columns))
	for _, column := range columns {
		labels = append(labels, column.Label)
	}
	return labels
}

func reportRowValues(row map[string]any, columns []SchoolReportColumn) []string {
	values := make([]string, 0, len(columns))
	for _, column := range columns {
		value := row[column.Key]
		switch typed := value.(type) {
		case nil:
			values = append(values, "")
		case time.Time:
			values = append(values, typed.UTC().Format(time.DateOnly))
		case bool:
			values = append(values, map[bool]string{true: "Da", false: "Nu"}[typed])
		default:
			values = append(values, fmt.Sprint(typed))
		}
	}
	return values
}
