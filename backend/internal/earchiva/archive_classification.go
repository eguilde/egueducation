package earchiva

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
)

// ArchiveReviewPermission is intentionally separate from earchiva.manage:
// tenants can delegate classification confirmation without upload/delete power.
const ArchiveReviewPermission = "earchiva.review"

const archiveClassificationRuleSource = "romanian_school_archive_ocr_rules_v1"

type ClassificationFieldSuggestion struct {
	Value      string  `json:"value,omitempty"`
	Source     string  `json:"source"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence,omitempty"`
}

type ArchiveClassificationSuggestion struct {
	Category       ClassificationFieldSuggestion `json:"category"`
	Fond           ClassificationFieldSuggestion `json:"fond"`
	Series         ClassificationFieldSuggestion `json:"series"`
	DocumentType   ClassificationFieldSuggestion `json:"document_type"`
	DocumentDate   ClassificationFieldSuggestion `json:"document_date"`
	DocumentNumber ClassificationFieldSuggestion `json:"document_number"`
}

type ArchiveClassificationReview struct {
	ID                   string                          `json:"id"`
	DocumentID           string                          `json:"document_id"`
	VersionID            string                          `json:"version_id"`
	State                string                          `json:"state"`
	Revision             int                             `json:"revision"`
	Suggestion           ArchiveClassificationSuggestion `json:"suggestion"`
	SuggestionConfidence float64                         `json:"suggestion_confidence"`
	SuggestionSource     string                          `json:"suggestion_source"`
	RequiresHumanReview  bool                            `json:"requires_human_review"`
	GeneratedAt          string                          `json:"generated_at"`
	ReviewedAt           string                          `json:"reviewed_at,omitempty"`
	ReviewedBy           string                          `json:"reviewed_by,omitempty"`
	FinalClassification  map[string]any                  `json:"final_classification,omitempty"`
	ReviewNote           string                          `json:"review_note,omitempty"`
}

type ArchiveClassificationReviewPage struct {
	Items    []ArchiveClassificationReview `json:"items"`
	Page     int                           `json:"page"`
	PageSize int                           `json:"page_size"`
	Total    int64                         `json:"total"`
}

// ClassificationInput is the worker integration contract. The worker must
// call UpsertSuggestion after OCR succeeds, passing the version it processed.
// It must never write a suggestion into archive_documents.metadata.
type ClassificationInput struct {
	InstitutionID string
	DocumentID    string
	VersionID     string
	OCRText       string
}

type ClassificationReviewService struct{ pool *appdb.SessionPool }

type classificationQueryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewClassificationReviewService(pool *appdb.SessionPool) *ClassificationReviewService {
	return &ClassificationReviewService{pool: pool}
}

func (s *ClassificationReviewService) UpsertSuggestion(ctx context.Context, input ClassificationInput) (*ArchiveClassificationReview, error) {
	if s == nil || s.pool == nil || strings.TrimSpace(input.InstitutionID) == "" || !validUUID(input.DocumentID) || !validUUID(input.VersionID) {
		return nil, errors.New("invalid archive classification input")
	}
	return upsertClassificationSuggestion(ctx, s.pool, input)
}

func (s *ClassificationReviewService) UpsertSuggestionTx(ctx context.Context, tx pgx.Tx, input ClassificationInput) (*ArchiveClassificationReview, error) {
	if s == nil || tx == nil || strings.TrimSpace(input.InstitutionID) == "" || !validUUID(input.DocumentID) || !validUUID(input.VersionID) {
		return nil, errors.New("invalid archive classification input")
	}
	return upsertClassificationSuggestion(ctx, tx, input)
}

func upsertClassificationSuggestion(ctx context.Context, target classificationQueryRower, input ClassificationInput) (*ArchiveClassificationReview, error) {
	suggestion := SuggestRomanianSchoolArchiveClassification(input.OCRText)
	payload, err := json.Marshal(suggestion)
	if err != nil {
		return nil, err
	}
	confidence := classificationConfidence(suggestion)
	state := "pending_review"
	if confidence < 0.70 {
		state = "needs_review"
	}

	var review ArchiveClassificationReview
	err = target.QueryRow(ctx, `
		insert into archive_document_classification_reviews (
			institution_id, document_id, version_id, state, suggestion,
			suggestion_confidence, suggestion_source, requires_human_review, generated_at, updated_at
		) values ($1, $2::uuid, $3::uuid, $4, $5::jsonb, $6, $7, true, now(), now())
		on conflict (institution_id, document_id, version_id) do update
		set state = excluded.state, suggestion = excluded.suggestion,
			suggestion_confidence = excluded.suggestion_confidence,
			suggestion_source = excluded.suggestion_source,
			requires_human_review = true, generated_at = now(), updated_at = now(),
			revision = archive_document_classification_reviews.revision + 1,
			reviewed_at = null, reviewed_by = '', final_classification = '{}'::jsonb, review_note = ''
		where archive_document_classification_reviews.state in ('pending_review', 'needs_review')
		returning id::text, document_id::text, version_id::text, state, revision, suggestion,
			suggestion_confidence, suggestion_source, requires_human_review,
			to_char(generated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`, input.InstitutionID, input.DocumentID, input.VersionID, state, string(payload), confidence, archiveClassificationRuleSource).Scan(
		&review.ID, &review.DocumentID, &review.VersionID, &review.State, &review.Revision, &review.Suggestion,
		&review.SuggestionConfidence, &review.SuggestionSource, &review.RequiresHumanReview, &review.GeneratedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} // a human review is closed and immutable
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// ListPending handles GET /api/earchiva/classification-reviews?state=pending_review|needs_review&page=1&pageSize=25.
func (s *ClassificationReviewService) ListPending(w http.ResponseWriter, r *http.Request) {
	institutionID, ok := classificationInstitution(w, r)
	if !ok {
		return
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if state == "" {
		state = "pending_review"
	}
	if state != "pending_review" && state != "needs_review" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_classification_review_state"})
		return
	}
	page, size := archiveAdminPage(r.URL.Query().Get("page"), 1, 10000), archiveAdminPage(r.URL.Query().Get("pageSize"), 25, 100)
	var total int64
	if err := s.pool.QueryRow(r.Context(), `select count(*) from archive_document_classification_reviews where institution_id=$1 and state=$2`, institutionID, state).Scan(&total); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "archive_classification_reviews_failed"})
		return
	}
	rows, err := s.pool.Query(r.Context(), classificationReviewSelect+` where institution_id=$1 and state=$2 order by generated_at asc, id asc limit $3 offset $4`, institutionID, state, size, (page-1)*size)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "archive_classification_reviews_failed"})
		return
	}
	defer rows.Close()
	items := make([]ArchiveClassificationReview, 0, size)
	for rows.Next() {
		var item ArchiveClassificationReview
		if err := scanClassificationReview(rows, &item); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "archive_classification_reviews_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "archive_classification_reviews_failed"})
		return
	}
	httpx.JSON(w, 200, ArchiveClassificationReviewPage{Items: items, Page: page, PageSize: size, Total: total})
}

type ArchiveFinalClassification struct {
	Category       string `json:"category"`
	Fond           string `json:"fond"`
	Series         string `json:"series"`
	DocumentType   string `json:"document_type"`
	DocumentDate   string `json:"document_date,omitempty"`
	DocumentNumber string `json:"document_number,omitempty"`
}

type ArchiveClassificationApprovalRequest struct {
	Revision int    `json:"revision"`
	Note     string `json:"note"`
}

type ArchiveClassificationCorrectionRequest struct {
	Revision       int                        `json:"revision"`
	Classification ArchiveFinalClassification `json:"classification"`
	Note           string                     `json:"note"`
}

// Approve handles POST /api/earchiva/classification-reviews/{reviewID}/approve.
// The server uses the stored suggestion; callers cannot replace it here.
func (s *ClassificationReviewService) Approve(w http.ResponseWriter, r *http.Request) {
	s.decide(w, r, "approved")
}

// Correct handles POST /api/earchiva/classification-reviews/{reviewID}/correct.
func (s *ClassificationReviewService) Correct(w http.ResponseWriter, r *http.Request) {
	s.decide(w, r, "corrected")
}

func (s *ClassificationReviewService) decide(w http.ResponseWriter, r *http.Request, targetState string) {
	institutionID, ok := classificationInstitution(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "reviewID"))
	if !validUUID(id) {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_classification_review_id"})
		return
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10))
	decoder.DisallowUnknownFields()
	var revision int
	var note string
	var correction *ArchiveFinalClassification
	if targetState == "corrected" {
		var req ArchiveClassificationCorrectionRequest
		if err := decoder.Decode(&req); err != nil {
			httpx.JSON(w, 400, map[string]any{"code": "invalid_classification_review_payload"})
			return
		}
		revision, note, correction = req.Revision, req.Note, &req.Classification
	} else {
		var req ArchiveClassificationApprovalRequest
		if err := decoder.Decode(&req); err != nil {
			httpx.JSON(w, 400, map[string]any{"code": "invalid_classification_review_payload"})
			return
		}
		revision, note = req.Revision, req.Note
	}
	if revision < 1 {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_classification_review_payload"})
		return
	}
	note = strings.TrimSpace(note)
	if targetState == "corrected" && (correction == nil || !validFinalClassification(*correction)) {
		httpx.JSON(w, 400, map[string]any{"code": "classification_correction_required"})
		return
	}
	actor := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if actor == "" {
		httpx.JSON(w, 403, map[string]any{"code": "missing_actor_subject"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "archive_classification_review_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck
	var review ArchiveClassificationReview
	err = tx.QueryRow(r.Context(), classificationReviewSelect+` where id=$1::uuid and institution_id=$2 and revision=$3 and state in ('pending_review','needs_review') for update`, id, institutionID, revision).Scan(scanClassificationReviewArgs(&review)...)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, 409, map[string]any{"code": "classification_review_conflict_or_unavailable"})
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "archive_classification_review_failed"})
		return
	}
	final := finalClassificationFromSuggestion(review.Suggestion)
	if targetState == "corrected" {
		final = *correction
	}
	if !validFinalClassification(final) {
		httpx.JSON(w, 400, map[string]any{"code": "classification_correction_required"})
		return
	}
	payload, err := json.Marshal(final)
	if err != nil {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_classification_correction"})
		return
	}
	previousState := review.State
	err = tx.QueryRow(r.Context(), `update archive_document_classification_reviews set state=$1, revision=revision+1, reviewed_at=now(), reviewed_by=$2, final_classification=$3::jsonb, review_note=$4, updated_at=now() where id=$5::uuid and institution_id=$6 returning revision, to_char(reviewed_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"')`, targetState, actor, string(payload), note, id, institutionID).Scan(&review.Revision, &review.ReviewedAt)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "archive_classification_review_failed"})
		return
	}
	review.State = targetState
	review.ReviewedBy = actor
	finalJSON, _ := json.Marshal(final)
	_ = json.Unmarshal(finalJSON, &review.FinalClassification)
	review.ReviewNote = note
	if err := audit.Log(r.Context(), tx, audit.Event{ActorSubject: actor, Action: "earchiva.classification_review." + targetState, TargetType: "archive_document_classification_review", TargetID: review.ID, Summary: "OCR classification review " + targetState + ".", Details: map[string]any{"institution_id": institutionID, "document_id": review.DocumentID, "version_id": review.VersionID, "from_state": previousState, "to_state": targetState, "revision": review.Revision}}); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "archive_classification_review_failed"})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "archive_classification_review_failed"})
		return
	}
	httpx.JSON(w, 200, review)
}

func finalClassificationFromSuggestion(s ArchiveClassificationSuggestion) ArchiveFinalClassification {
	return ArchiveFinalClassification{Category: s.Category.Value, Fond: s.Fond.Value, Series: s.Series.Value, DocumentType: s.DocumentType.Value, DocumentDate: s.DocumentDate.Value, DocumentNumber: s.DocumentNumber.Value}
}

func validFinalClassification(value ArchiveFinalClassification) bool {
	fields := []string{value.Category, value.Fond, value.Series, value.DocumentType, value.DocumentDate, value.DocumentNumber}
	for _, field := range fields {
		if len([]rune(strings.TrimSpace(field))) > 200 {
			return false
		}
	}
	return strings.TrimSpace(value.Category) != "" && strings.TrimSpace(value.Fond) != "" && strings.TrimSpace(value.Series) != "" && strings.TrimSpace(value.DocumentType) != ""
}

const classificationReviewSelect = `select id::text, document_id::text, version_id::text, state, revision, suggestion, suggestion_confidence, suggestion_source, requires_human_review, to_char(generated_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'), coalesce(to_char(reviewed_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),''), reviewed_by, final_classification, review_note from archive_document_classification_reviews`

type classificationRow interface{ Scan(...any) error }

func scanClassificationReview(row classificationRow, review *ArchiveClassificationReview) error {
	return row.Scan(scanClassificationReviewArgs(review)...)
}
func scanClassificationReviewArgs(review *ArchiveClassificationReview) []any {
	return []any{&review.ID, &review.DocumentID, &review.VersionID, &review.State, &review.Revision, &review.Suggestion, &review.SuggestionConfidence, &review.SuggestionSource, &review.RequiresHumanReview, &review.GeneratedAt, &review.ReviewedAt, &review.ReviewedBy, &review.FinalClassification, &review.ReviewNote}
}
func classificationInstitution(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r))
	if id == "" {
		httpx.JSON(w, 403, map[string]any{"code": "missing_institution_context"})
		return "", false
	}
	return id, true
}
func validUUID(raw string) bool { _, err := uuid.Parse(raw); return err == nil }

var datePattern = regexp.MustCompile(`\b(0?[1-9]|[12][0-9]|3[01])[.\-/](0?[1-9]|1[0-2])[.\-/]((?:19|20)\d{2})\b`)
var numberPattern = regexp.MustCompile(`(?i)\b(?:nr\.?|num[aă]rul)\s*([A-Z]{0,4}[-/]?\d{1,8}(?:[-/]\d{1,4})?)\b`)

// SuggestRomanianSchoolArchiveClassification is deterministic and explainable;
// it makes no network/model calls and produces no final classification.
func SuggestRomanianSchoolArchiveClassification(ocrText string) ArchiveClassificationSuggestion {
	normalized := strings.ToLower(strings.Join(strings.Fields(ocrText), " "))
	unknown := func() ClassificationFieldSuggestion {
		return ClassificationFieldSuggestion{Source: archiveClassificationRuleSource, Confidence: 0}
	}
	result := ArchiveClassificationSuggestion{Category: unknown(), Fond: unknown(), Series: unknown(), DocumentType: unknown(), DocumentDate: unknown(), DocumentNumber: unknown()}
	type rule struct {
		needles                     []string
		category, fond, series, typ string
		confidence                  float64
	}
	rules := []rule{
		{[]string{"catalog școlar", "catalog scolar", "situația școlară", "situatia scolara"}, "elevi", "fond_institutional_scoala", "evidenta_elevi", "catalog_scolar", .93},
		{[]string{"contract individual de muncă", "contract individual de munca", "dosar personal"}, "personal", "fond_institutional_scoala", "personal", "dosar_personal", .92},
		{[]string{"hotărârea consiliului de administrație", "hotararea consiliului de administratie", "consiliul de administrație", "consiliul de administratie"}, "guvernanta", "fond_institutional_scoala", "consiliu_administratie", "hotarare_consiliu_administratie", .91},
		{[]string{"proces-verbal", "proces verbal"}, "guvernanta", "fond_institutional_scoala", "procese_verbale", "proces_verbal", .83},
		{[]string{"decizie", "decizia"}, "guvernanta", "fond_institutional_scoala", "decizii", "decizie", .79},
		{[]string{"factură", "factura", "ordin de plată", "ordin de plata", "buget"}, "financiar", "fond_institutional_scoala", "financiar_contabil", "document_financiar", .82},
		{[]string{"adeverință", "adeverinta"}, "elevi", "fond_institutional_scoala", "evidenta_elevi", "adeverinta", .75},
	}
	for _, rule := range rules {
		for _, needle := range rule.needles {
			if strings.Contains(normalized, needle) {
				evidence := needle
				mk := func(value string) ClassificationFieldSuggestion {
					return ClassificationFieldSuggestion{Value: value, Source: archiveClassificationRuleSource, Confidence: rule.confidence, Evidence: evidence}
				}
				result.Category = mk(rule.category)
				result.Fond = mk(rule.fond)
				result.Series = mk(rule.series)
				result.DocumentType = mk(rule.typ)
				goto extraction
			}
		}
	}
extraction:
	if match := datePattern.FindStringSubmatch(normalized); len(match) == 4 {
		result.DocumentDate = ClassificationFieldSuggestion{Value: match[3] + "-" + pad2(match[2]) + "-" + pad2(match[1]), Source: archiveClassificationRuleSource, Confidence: .86, Evidence: match[0]}
	}
	if match := numberPattern.FindStringSubmatch(ocrText); len(match) == 2 {
		result.DocumentNumber = ClassificationFieldSuggestion{Value: strings.ToUpper(match[1]), Source: archiveClassificationRuleSource, Confidence: .84, Evidence: match[0]}
	}
	return result
}
func pad2(raw string) string {
	if len(raw) == 1 {
		return "0" + raw
	}
	return raw
}
func classificationConfidence(s ArchiveClassificationSuggestion) float64 {
	values := []float64{s.Category.Confidence, s.Fond.Confidence, s.Series.Confidence, s.DocumentType.Confidence}
	sort.Float64s(values)
	return (values[1] + values[2] + values[3]) / 3
}
