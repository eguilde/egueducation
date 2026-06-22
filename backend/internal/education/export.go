package education

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/eguilde/egueducation/internal/httpx"
)

type ExportRequest struct {
	Title    string     `json:"title"`
	Filename string     `json:"filename"`
	Headers  []string   `json:"headers"`
	Rows     [][]string `json:"rows"`
}

func (s *Service) ExportPDF(w http.ResponseWriter, r *http.Request) {
	req, ok := parseExportRequest(w, r)
	if !ok {
		return
	}

	lines := make([]string, 0, len(req.Rows)+2)
	lines = append(lines, strings.Join(req.Headers, " | "))
	lines = append(lines, strings.Repeat("-", minInt(120, maxInt(12, len(lines[0])))))
	for _, row := range req.Rows {
		lines = append(lines, strings.Join(row, " | "))
	}

	writeEducationPDFDownload(w, req.Title, sanitizeExportFilename(req.Filename, "education-export"), lines)
}

func (s *Service) ExportCSV(w http.ResponseWriter, r *http.Request) {
	req, ok := parseExportRequest(w, r)
	if !ok {
		return
	}

	var csvBuffer bytes.Buffer
	csvBuffer.WriteString("\uFEFF")
	writer := csv.NewWriter(&csvBuffer)
	_ = writer.Write(req.Headers)
	for _, row := range req.Rows {
		_ = writer.Write(row)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_export_csv_failed"})
		return
	}

	filename := sanitizeExportFilename(req.Filename, "education-export")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(csvBuffer.Bytes())
}

func parseExportRequest(w http.ResponseWriter, r *http.Request) (ExportRequest, bool) {
	var req ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_education_export_payload"})
		return ExportRequest{}, false
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Filename = strings.TrimSpace(req.Filename)
	if req.Title == "" {
		req.Title = "Export educational"
	}
	if len(req.Headers) == 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_education_export_headers"})
		return ExportRequest{}, false
	}
	if len(req.Rows) == 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_education_export_rows"})
		return ExportRequest{}, false
	}
	if len(req.Rows) > 5000 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "education_export_limit_exceeded"})
		return ExportRequest{}, false
	}
	for _, row := range req.Rows {
		if len(row) != len(req.Headers) {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_education_export_matrix"})
			return ExportRequest{}, false
		}
	}

	return req, true
}

func sanitizeExportFilename(value string, fallback string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return fallback
	}

	replacer := strings.NewReplacer(
		" ", "-",
		"/", "-",
		"\\", "-",
		":", "-",
		";", "-",
		",", "-",
		".", "-",
		"(", "",
		")", "",
	)
	value = replacer.Replace(value)
	value = strings.Trim(value, "-")
	if value == "" {
		return fallback
	}
	return value
}

func writeEducationPDFDownload(w http.ResponseWriter, title string, filename string, lines []string) {
	pdf := buildEducationPDF(title, lines)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, sanitizeExportFilename(filename, "education-export")))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}

func buildEducationPDF(title string, lines []string) []byte {
	wrappedLines := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		wrappedLines = append(wrappedLines, wrapEducationPDFLine(line, 100)...)
	}

	const linesPerPage = 55
	pageCount := maxInt(1, (len(wrappedLines)+linesPerPage-1)/linesPerPage)

	streams := make([]string, 0, pageCount)
	for pageIndex := 0; pageIndex < pageCount; pageIndex++ {
		start := pageIndex * linesPerPage
		end := minInt(start+linesPerPage, len(wrappedLines))

		var content bytes.Buffer
		writeLine := func(text string, x, y int) {
			fmt.Fprintf(&content, "BT /F1 9 Tf %d %d Td (%s) Tj ET\n", x, y, escapeEducationPDFText(text))
		}

		pageTitle := title
		if pageCount > 1 {
			pageTitle = fmt.Sprintf("%s (%d/%d)", title, pageIndex+1, pageCount)
		}
		writeLine(pageTitle, 48, 790)
		y := 768
		for _, line := range wrappedLines[start:end] {
			writeLine(line, 48, y)
			y -= 13
		}
		streams = append(streams, content.String())
	}

	var pdf bytes.Buffer
	totalObjects := 3 + pageCount*2
	offsets := make([]int, 0, totalObjects)
	writeObj := func(obj string) {
		offsets = append(offsets, pdf.Len())
		pdf.WriteString(obj)
	}

	pdf.WriteString("%PDF-1.4\n")
	writeObj("1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n")
	kids := make([]string, 0, pageCount)
	for pageIndex := 0; pageIndex < pageCount; pageIndex++ {
		pageObjectID := 4 + pageIndex*2
		kids = append(kids, fmt.Sprintf("%d 0 R", pageObjectID))
	}
	writeObj(fmt.Sprintf("2 0 obj << /Type /Pages /Kids [%s] /Count %d >> endobj\n", strings.Join(kids, " "), pageCount))
	writeObj("3 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> endobj\n")

	for pageIndex, stream := range streams {
		pageObjectID := 4 + pageIndex*2
		streamObjectID := pageObjectID + 1
		writeObj(fmt.Sprintf("%d 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >> endobj\n", pageObjectID, streamObjectID))
		writeObj(fmt.Sprintf("%d 0 obj << /Length %d >> stream\n%s\nendstream endobj\n", streamObjectID, len(stream), stream))
	}

	xrefStart := pdf.Len()
	pdf.WriteString(fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", totalObjects+1))
	for _, offset := range offsets {
		pdf.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}
	pdf.WriteString(fmt.Sprintf("trailer << /Size %d /Root 1 0 R >>\nstartxref\n", totalObjects+1))
	pdf.WriteString(fmt.Sprintf("%d\n%%EOF\n", xrefStart))
	return pdf.Bytes()
}

func escapeEducationPDFText(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "(", "\\(")
	text = strings.ReplaceAll(text, ")", "\\)")
	return text
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func wrapEducationPDFLine(line string, maxLength int) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return []string{""}
	}
	if len(line) <= maxLength {
		return []string{line}
	}

	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{line}
	}

	lines := make([]string, 0, len(words)/2+1)
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		if len(candidate) <= maxLength {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = word
	}
	lines = append(lines, current)
	return lines
}
