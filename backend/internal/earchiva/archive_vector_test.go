package earchiva

import "testing"

func TestBuildArchiveEmbeddingStable(t *testing.T) {
	left := buildArchiveEmbedding("Fiscal decision for school transport")
	right := buildArchiveEmbedding("Fiscal decision for school transport")
	if len(left) != archiveEmbeddingDimensions {
		t.Fatalf("expected %d dimensions, got %d", archiveEmbeddingDimensions, len(left))
	}
	if len(right) != archiveEmbeddingDimensions {
		t.Fatalf("expected %d dimensions, got %d", archiveEmbeddingDimensions, len(right))
	}
	if got := archiveVectorDot(left, right); got < 0.99 {
		t.Fatalf("expected stable embeddings, got dot=%.4f", got)
	}
}

func TestBuildArchiveEmbeddingPrefersRelatedText(t *testing.T) {
	related := archiveVectorDot(
		buildArchiveEmbedding("employee disciplinary decision appeal"),
		buildArchiveEmbedding("disciplinary decision for employee appeal"),
	)
	unrelated := archiveVectorDot(
		buildArchiveEmbedding("employee disciplinary decision appeal"),
		buildArchiveEmbedding("menu recipe pasta tomato basil"),
	)
	if related <= unrelated {
		t.Fatalf("expected related text to score higher than unrelated text: related=%.4f unrelated=%.4f", related, unrelated)
	}
}

func TestNormalizeArchiveSearchMode(t *testing.T) {
	if got := normalizeArchiveSearchMode("VECTOR"); got != archiveVectorSearchModeVector {
		t.Fatalf("expected vector mode, got %q", got)
	}
	if got := normalizeArchiveSearchMode("hybrid"); got != archiveVectorSearchModeHybrid {
		t.Fatalf("expected hybrid mode, got %q", got)
	}
	if got := normalizeArchiveSearchMode(""); got != archiveVectorSearchModeFts {
		t.Fatalf("expected default fts mode, got %q", got)
	}
}
