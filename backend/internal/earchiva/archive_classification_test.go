package earchiva

import (
	"math"
	"testing"
)

func TestSuggestRomanianSchoolArchiveClassificationCatalog(t *testing.T) {
	s := SuggestRomanianSchoolArchiveClassification("CATALOG ȘCOLAR clasa a VIII-a Nr. 42 din 03.09.2025")
	if s.Category.Value != "elevi" || s.Series.Value != "evidenta_elevi" || s.DocumentType.Value != "catalog_scolar" {
		t.Fatalf("unexpected school catalog suggestion: %#v", s)
	}
	if s.DocumentDate.Value != "2025-09-03" || s.DocumentNumber.Value != "42" {
		t.Fatalf("unexpected extracted values: %#v", s)
	}
	if s.Category.Source != archiveClassificationRuleSource || s.Category.Confidence < .9 {
		t.Fatalf("suggestion must carry deterministic provenance/confidence: %#v", s.Category)
	}
}

func TestSuggestRomanianSchoolArchiveClassificationLowConfidenceNeedsReview(t *testing.T) {
	s := SuggestRomanianSchoolArchiveClassification("scan ilizibil, pagină fără antet")
	if got := classificationConfidence(s); got != 0 {
		t.Fatalf("want zero confidence, got %v", got)
	}
	if s.DocumentType.Value != "" || s.Category.Source != archiveClassificationRuleSource {
		t.Fatalf("unknown OCR must remain explicitly unknown: %#v", s)
	}
}

func TestSuggestRomanianSchoolArchiveClassificationPersonal(t *testing.T) {
	s := SuggestRomanianSchoolArchiveClassification("DOSAR PERSONAL - Contract individual de muncă nr. AB-99/2024")
	if s.Category.Value != "personal" || s.DocumentType.Value != "dosar_personal" {
		t.Fatalf("unexpected personnel suggestion: %#v", s)
	}
	if s.DocumentNumber.Value != "AB-99/2024" {
		t.Fatalf("number extraction lost: %#v", s.DocumentNumber)
	}
}

func TestClassificationConfidenceDoesNotTreatDateAsClassification(t *testing.T) {
	s := ArchiveClassificationSuggestion{Category: ClassificationFieldSuggestion{Confidence: .8}, Fond: ClassificationFieldSuggestion{Confidence: .7}, Series: ClassificationFieldSuggestion{Confidence: .6}, DocumentType: ClassificationFieldSuggestion{Confidence: .5}, DocumentDate: ClassificationFieldSuggestion{Confidence: 1}}
	if got := classificationConfidence(s); math.Abs(got-.7) > 0.000001 {
		t.Fatalf("want 0.7 based on taxonomy fields only, got %v", got)
	}
}
