package nomenclature

import appdb "github.com/eguilde/egueducation/internal/db"

type Service struct {
	pool *appdb.SessionPool
}

func NewService(pool *appdb.SessionPool) *Service {
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
