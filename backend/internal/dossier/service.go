package dossier

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Purpose string

const (
	PurposeReadiness Purpose = "readiness"
	PurposeSubmit    Purpose = "submit"
	PurposeApprove   Purpose = "approve"
)

type RelationCounts struct {
	Total        int
	Primary      int
	Supporting   int
	Decision     int
	ArchiveBasis int
	GDPRBasis    int
}

type Requirement struct {
	ID                   string `json:"id"`
	SourceModule         string `json:"source_module"`
	RelationType         string `json:"relation_type"`
	MinCount             int    `json:"min_count"`
	RequiredForReadiness bool   `json:"required_for_readiness"`
	RequiredForSubmit    bool   `json:"required_for_submit"`
	RequiredForApprove   bool   `json:"required_for_approve"`
}

type Queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func Evaluate(ctx context.Context, pool Queryer, sourceModule string, hasSourceRecord bool, counts RelationCounts, purpose Purpose) (bool, []string, error) {
	if !hasSourceRecord {
		return true, []string{}, nil
	}

	requirements, err := ListRequirements(ctx, pool, sourceModule)
	if err != nil {
		return false, nil, err
	}

	filtered := filterRequirements(requirements, purpose)
	if len(filtered) == 0 {
		if counts.Total == 0 {
			return false, []string{"supporting"}, nil
		}
		return true, []string{}, nil
	}

	missing := make([]string, 0, len(filtered))
	for _, requirement := range filtered {
		if relationCount(counts, requirement.RelationType) < requirement.MinCount {
			missing = append(missing, requirement.RelationType)
		}
	}

	return len(missing) == 0, missing, nil
}

func ListRequirements(ctx context.Context, pool Queryer, sourceModule string) ([]Requirement, error) {
	rows, err := pool.Query(ctx, `
		select
			id::text,
			source_module,
			relation_type,
			min_count,
			required_for_readiness,
			required_for_submit,
			required_for_approve
		from workflow_dossier_requirements
		where source_module = $1
		order by source_module, relation_type
	`, sourceModule)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requirements := []Requirement{}
	for rows.Next() {
		var requirement Requirement
		if err := rows.Scan(
			&requirement.ID,
			&requirement.SourceModule,
			&requirement.RelationType,
			&requirement.MinCount,
			&requirement.RequiredForReadiness,
			&requirement.RequiredForSubmit,
			&requirement.RequiredForApprove,
		); err != nil {
			return nil, err
		}
		requirements = append(requirements, requirement)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return requirements, nil
}

func CountSQL(prefix string) string {
	return fmt.Sprintf(`
		coalesce(%s.total_count, 0) as linked_documents_count,
		coalesce(%s.primary_count, 0) as primary_count,
		coalesce(%s.supporting_count, 0) as supporting_count,
		coalesce(%s.decision_count, 0) as decision_count,
		coalesce(%s.archive_basis_count, 0) as archive_basis_count,
		coalesce(%s.gdpr_basis_count, 0) as gdpr_basis_count
	`, prefix, prefix, prefix, prefix, prefix, prefix)
}

func LateralJoinSQL(alias string, sourceModuleExpr string, sourceRecordExpr string) string {
	return fmt.Sprintf(`
		left join lateral (
			select
				count(*) as total_count,
				count(*) filter (where relation_type = 'primary') as primary_count,
				count(*) filter (where relation_type = 'supporting') as supporting_count,
				count(*) filter (where relation_type = 'decision') as decision_count,
				count(*) filter (where relation_type = 'archive_basis') as archive_basis_count,
				count(*) filter (where relation_type = 'gdpr_basis') as gdpr_basis_count
			from registratura_document_links l
			where l.source_module = %s
				and l.source_record_id = %s
		) %s on true
	`, sourceModuleExpr, sourceRecordExpr, alias)
}

func filterRequirements(requirements []Requirement, purpose Purpose) []Requirement {
	filtered := []Requirement{}
	for _, requirement := range requirements {
		switch purpose {
		case PurposeReadiness:
			if requirement.RequiredForReadiness {
				filtered = append(filtered, requirement)
			}
		case PurposeSubmit:
			if requirement.RequiredForSubmit {
				filtered = append(filtered, requirement)
			}
		case PurposeApprove:
			if requirement.RequiredForApprove {
				filtered = append(filtered, requirement)
			}
		}
	}
	return filtered
}

func relationCount(counts RelationCounts, relationType string) int {
	switch relationType {
	case "primary":
		return counts.Primary
	case "supporting":
		return counts.Supporting
	case "decision":
		return counts.Decision
	case "archive_basis":
		return counts.ArchiveBasis
	case "gdpr_basis":
		return counts.GDPRBasis
	default:
		return 0
	}
}
