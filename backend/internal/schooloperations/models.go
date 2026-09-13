package schooloperations

// Contract is the closed server-side projection used by the list endpoint.
type Contract struct {
	ID              string  `json:"id"`
	SupplierPartyID string  `json:"supplier_party_id"`
	SupplierName    string  `json:"supplier_name"`
	ContractNumber  string  `json:"contract_number"`
	Title           string  `json:"title"`
	Category        string  `json:"category"`
	LifecycleStatus string  `json:"lifecycle_status"`
	StartsOn        *string `json:"starts_on,omitempty"`
	EndsOn          *string `json:"ends_on,omitempty"`
	TotalValue      float64 `json:"total_value"`
	Currency        string  `json:"currency"`
	ExpectedVersion int     `json:"expected_version"`
	ArchiveStatus   string  `json:"archive_status"`
}

type SupplierOption struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
	TaxID       string `json:"tax_id"`
}

// CreateContractRequest deliberately has no scope or policy fields.
type CreateContractRequest struct {
	SupplierPartyID string  `json:"supplier_party_id"`
	ContractNumber  string  `json:"contract_number"`
	Title           string  `json:"title"`
	Category        string  `json:"category"`
	StartsOn        string  `json:"starts_on"`
	EndsOn          string  `json:"ends_on"`
	TotalValue      float64 `json:"total_value"`
	Currency        string  `json:"currency"`
	IdempotencyKey  string  `json:"idempotency_key"`
}

type TransitionContractRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	Status          string `json:"status"`
}

// AmendContractRequest changes only a non-final contract. The expected version
// is mandatory to prevent a browser from overwriting another user's change.
type AmendContractRequest struct {
	ExpectedVersion int     `json:"expected_version"`
	Title           string  `json:"title"`
	Category        string  `json:"category"`
	StartsOn        string  `json:"starts_on"`
	EndsOn          string  `json:"ends_on"`
	TotalValue      float64 `json:"total_value"`
	Currency        string  `json:"currency"`
}

type ContractObligation struct {
	ID              string   `json:"id"`
	ContractID      string   `json:"contract_id"`
	Title           string   `json:"title"`
	DueOn           *string  `json:"due_on,omitempty"`
	Status          string   `json:"status"`
	SLAHours        *int     `json:"sla_hours,omitempty"`
	GuaranteeValue  *float64 `json:"guarantee_value,omitempty"`
	ExpectedVersion int      `json:"expected_version"`
}

type CreateContractObligationRequest struct {
	Title          string   `json:"title"`
	DueOn          string   `json:"due_on"`
	SLAHours       *int     `json:"sla_hours"`
	GuaranteeValue *float64 `json:"guarantee_value"`
}
