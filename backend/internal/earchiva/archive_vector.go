package earchiva

import (
	"fmt"
	"hash/fnv"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	archiveEmbeddingDimensions    = 256
	archiveVectorSearchModeFts    = "fts"
	archiveVectorSearchModeVector = "vector"
	archiveVectorSearchModeHybrid = "hybrid"
	archiveVectorSearchThreshold  = 0.12
	archiveHybridFTSWeight        = 0.65
	archiveHybridVectorWeight     = 0.35
)

var archiveEmbeddingTokenRe = regexp.MustCompile(`(?i)[\pL\pN]+`)

func normalizeArchiveSearchMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case archiveVectorSearchModeVector:
		return archiveVectorSearchModeVector
	case archiveVectorSearchModeHybrid:
		return archiveVectorSearchModeHybrid
	default:
		return archiveVectorSearchModeFts
	}
}

func archiveDocumentEmbeddingInput(info archiveIngestionContext, text string) string {
	parts := []string{
		strings.TrimSpace(info.Title),
		strings.TrimSpace(info.OriginalFileName),
		strings.TrimSpace(info.SourceKind),
		strings.TrimSpace(info.SourceSystem),
		strings.TrimSpace(info.ExternalReference),
	}
	if info.TaxonomyCode != nil {
		parts = append(parts, strings.TrimSpace(*info.TaxonomyCode))
	}
	if info.TaxonomyLabel != nil {
		parts = append(parts, strings.TrimSpace(*info.TaxonomyLabel))
	}
	if info.DocumentDate != nil {
		parts = append(parts, strings.TrimSpace(*info.DocumentDate))
	}
	if len(info.Metadata) > 0 {
		parts = append(parts, archiveMetadataText(info.Metadata))
	}
	parts = append(parts, strings.TrimSpace(text))
	return strings.Join(filterArchiveEmpty(parts), " ")
}

func archiveMetadataText(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(metadata)*2)
	for _, key := range keys {
		value := metadata[key]
		if key = strings.TrimSpace(key); key != "" {
			parts = append(parts, key)
		}
		if text := strings.TrimSpace(archiveAnyToText(value)); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func archiveAnyToText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []string:
		return strings.Join(filterArchiveEmpty(typed), " ")
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(archiveAnyToText(item)); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, " ")
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(typed)*2)
		for _, key := range keys {
			item := typed[key]
			if key = strings.TrimSpace(key); key != "" {
				parts = append(parts, key)
			}
			if text := strings.TrimSpace(archiveAnyToText(item)); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, " ")
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func filterArchiveEmpty(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func buildArchiveEmbedding(text string) []float64 {
	vector := make([]float64, archiveEmbeddingDimensions)
	tokens := archiveEmbeddingTokenRe.FindAllString(strings.ToLower(text), -1)
	if len(tokens) == 0 {
		return vector
	}

	for index, token := range tokens {
		if len(token) < 2 {
			continue
		}
		archiveAddEmbeddingFeature(vector, token, 1.0)
		if len(token) >= 8 {
			archiveAddEmbeddingFeature(vector, archiveTokenPrefix(token, 4), 0.25)
		}
		if index+1 < len(tokens) {
			archiveAddEmbeddingFeature(vector, token+"|"+tokens[index+1], 0.6)
		}
		if index+2 < len(tokens) {
			archiveAddEmbeddingFeature(vector, token+"|"+tokens[index+1]+"|"+tokens[index+2], 0.35)
		}
	}

	archiveNormalizeEmbedding(vector)
	return vector
}

func archiveAddEmbeddingFeature(vector []float64, token string, weight float64) {
	if token == "" || weight == 0 {
		return
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(token))
	hash := hasher.Sum64()
	index := int(hash % uint64(len(vector)))
	sign := 1.0
	if hash&1 == 1 {
		sign = -1.0
	}
	vector[index] += sign * weight
}

func archiveTokenPrefix(token string, runes int) string {
	if token == "" || runes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(token) <= runes {
		return token
	}
	var builder strings.Builder
	builder.Grow(len(token))
	count := 0
	for _, r := range token {
		if count >= runes {
			break
		}
		builder.WriteRune(r)
		count++
	}
	return builder.String()
}

func archiveNormalizeEmbedding(vector []float64) {
	var norm float64
	for _, value := range vector {
		norm += value * value
	}
	if norm == 0 {
		return
	}
	scale := 1 / math.Sqrt(norm)
	for index, value := range vector {
		vector[index] = value * scale
	}
}

func archiveVectorDot(left, right []float64) float64 {
	if len(left) == 0 || len(left) != len(right) {
		return 0
	}
	var score float64
	for index := range left {
		score += left[index] * right[index]
	}
	return score
}
