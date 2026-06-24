package registratura

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type resolvedDocumentParties struct {
	Correspondent        string
	AssignedTo           string
	CorrespondentPartyID *string
	AssignedPartyID      *string
}

func (s *Service) resolveDocumentParties(
	ctx context.Context,
	tx pgx.Tx,
	institutionID string,
	direction string,
	correspondent string,
	assignedTo string,
	correspondentPartyID *string,
	assignedPartyID *string,
) (resolvedDocumentParties, error) {
	resolved := resolvedDocumentParties{
		Correspondent: strings.TrimSpace(correspondent),
		AssignedTo:    strings.TrimSpace(assignedTo),
	}

	if correspondentPartyID != nil && strings.TrimSpace(*correspondentPartyID) != "" {
		party, err := lookupPartyLabelTx(ctx, tx, *correspondentPartyID, institutionID)
		if err != nil {
			return resolved, err
		}
		resolved.Correspondent = party
		resolved.CorrespondentPartyID = correspondentPartyID
	}
	if assignedPartyID != nil && strings.TrimSpace(*assignedPartyID) != "" {
		party, err := lookupPartyLabelTx(ctx, tx, *assignedPartyID, institutionID)
		if err != nil {
			return resolved, err
		}
		resolved.AssignedTo = party
		resolved.AssignedPartyID = assignedPartyID
	}

	defaultPartyID, defaultPartyLabel, err := lookupDefaultOrganizationPartyTx(ctx, tx, institutionID)
	if err != nil {
		return resolved, err
	}

	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "intrare":
		if resolved.AssignedTo == "" {
			resolved.AssignedTo = defaultPartyLabel
			resolved.AssignedPartyID = &defaultPartyID
		}
		if resolved.Correspondent == "" {
			return resolved, fmt.Errorf("missing correspondent for incoming document")
		}
	case "iesire":
		if resolved.Correspondent == "" {
			resolved.Correspondent = defaultPartyLabel
			resolved.CorrespondentPartyID = &defaultPartyID
		}
		if resolved.AssignedTo == "" {
			return resolved, fmt.Errorf("missing recipient for outgoing document")
		}
	case "intern":
		if resolved.Correspondent == "" {
			resolved.Correspondent = defaultPartyLabel
			resolved.CorrespondentPartyID = &defaultPartyID
		}
		if resolved.AssignedTo == "" {
			resolved.AssignedTo = defaultPartyLabel
			resolved.AssignedPartyID = &defaultPartyID
		}
	}

	if resolved.Correspondent == "" || resolved.AssignedTo == "" {
		return resolved, fmt.Errorf("missing document parties")
	}

	return resolved, nil
}

func lookupPartyLabelTx(ctx context.Context, tx pgx.Tx, id string, institutionID string) (string, error) {
	var label string
	if err := tx.QueryRow(ctx, `
		select display_name
		from app_parties
		where id::text = $1 and institution_id = $2 and active = true
	`, strings.TrimSpace(id), strings.TrimSpace(institutionID)).Scan(&label); err != nil {
		return "", err
	}
	return label, nil
}

func lookupDefaultOrganizationPartyTx(ctx context.Context, tx pgx.Tx, institutionID string) (string, string, error) {
	var id, label string
	if err := tx.QueryRow(ctx, `
		select id::text, display_name
		from app_parties
		where institution_id = $1 and is_default_organization = true and active = true
		order by updated_at desc, created_at desc
		limit 1
	`, strings.TrimSpace(institutionID)).Scan(&id, &label); err != nil {
		return "", "", err
	}
	return id, label, nil
}
