package nomenclature

import "github.com/jackc/pgx/v5/pgxpool"

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type Item struct {
	ID        string `json:"id"`
	Domain    string `json:"domain"`
	Code      string `json:"code"`
	LabelRO   string `json:"label_ro"`
	LabelEN   string `json:"label_en"`
	Active    bool   `json:"active"`
	SortOrder int    `json:"sort_order"`
}
