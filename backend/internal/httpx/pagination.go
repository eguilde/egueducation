package httpx

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type PageQuery struct {
	Page      int
	PageSize  int
	Sort      string
	Direction string
	Filters   map[string]string
}

type PageResponse[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

func ParsePageQuery(values url.Values, allowedSorts map[string]struct{}, filterKeys []string) PageQuery {
	page := parsePositiveInt(values.Get("page"), 1)
	pageSize := parsePositiveInt(values.Get("pageSize"), 25)
	if pageSize > 100 {
		pageSize = 100
	}

	sort := values.Get("sort")
	if _, ok := allowedSorts[sort]; !ok {
		sort = ""
	}

	direction := strings.ToLower(values.Get("direction"))
	if direction != "desc" {
		direction = "asc"
	}

	filters := make(map[string]string, len(filterKeys))
	for _, key := range filterKeys {
		value := strings.TrimSpace(values.Get("filter." + key))
		if value != "" {
			filters[key] = value
		}
	}

	return PageQuery{
		Page:      page,
		PageSize:  pageSize,
		Sort:      sort,
		Direction: direction,
		Filters:   filters,
	}
}

func WritePage[T any](w http.ResponseWriter, status int, items []T, total, page, pageSize int) {
	JSON(w, status, PageResponse[T]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
