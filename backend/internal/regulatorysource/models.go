package regulatorysource

import "time"

type RegulatorySource struct {
	ID                        string     `json:"id"`
	Citation                  string     `json:"citation"`
	PublisherURL              string     `json:"publisher_url"`
	Issuer                    string     `json:"issuer"`
	SourceKind                string     `json:"source_kind"`
	ApplicableFrom            *string    `json:"applicable_from"`
	ApplicableUntil           *string    `json:"applicable_until"`
	Status                    string     `json:"status"`
	ExpectedVersion           int        `json:"expected_version"`
	CreatedBySubject          string     `json:"created_by_subject"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`
	LatestEvidenceID          *string    `json:"latest_evidence_id"`
	LatestEvidenceSHA256      *string    `json:"latest_evidence_sha256"`
	LatestEvidenceRetrievedAt *time.Time `json:"latest_evidence_retrieved_at"`
	ActivationEvidenceID      *string    `json:"activation_evidence_id"`
	ActivatedBySubject        *string    `json:"activated_by_subject"`
	ActivatedAt               *time.Time `json:"activated_at"`
	Assessment                *string    `json:"assessment"`
}

type RegisterRegulatorySourceRequest struct {
	Citation        string  `json:"citation"`
	PublisherURL    string  `json:"publisher_url"`
	Issuer          string  `json:"issuer"`
	SourceKind      string  `json:"source_kind"`
	ApplicableFrom  string  `json:"applicable_from"`
	ApplicableUntil *string `json:"applicable_until,omitempty"`
}

type VerifyRegulatorySourceRequest struct {
	ExpectedVersion int `json:"expected_version"`
}

type ActivateRegulatorySourceRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	EvidenceID      string `json:"evidence_id"`
	Assessment      string `json:"assessment"`
}
