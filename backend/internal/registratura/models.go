package registratura

type Registru struct {
	ID           int64   `json:"id"`
	Nume         string  `json:"nume"`
	PrefixNr     string  `json:"prefix_nr"`
	NrInceput    int     `json:"nr_inceput"`
	NrCurent     string  `json:"nr_curent"`
	NrUrmator    string  `json:"nr_urmator"`
	DataResetare *string `json:"data_resetare,omitempty"`
	TipRegistru  string  `json:"tip_registru"`
	IsDefault    bool    `json:"isDefault"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type Document struct {
	ID              string  `json:"id"`
	RegistruID      *int64  `json:"registru_id"`
	RegistryNumber  string  `json:"registry_number"`
	Subject         string  `json:"subject"`
	DocumentType    string  `json:"document_type"`
	Direction       string  `json:"direction"`
	Status          string  `json:"status"`
	Correspondent   string  `json:"correspondent"`
	AssignedTo      string  `json:"assigned_to"`
	CorrespondentPartyID *string `json:"correspondent_party_id,omitempty"`
	AssignedPartyID      *string `json:"assigned_party_id,omitempty"`
	InstitutionID   string  `json:"institution_id"`
	Confidentiality string  `json:"confidentiality"`
	Summary         string  `json:"summary"`
	RegisteredAt    string  `json:"registered_at"`
	DueDate         *string `json:"due_date"`
}

type Party struct {
	ID                    string `json:"id"`
	Code                  string `json:"code"`
	PartyType             string `json:"party_type"`
	DisplayName           string `json:"display_name"`
	ShortName             string `json:"short_name"`
	FirstName             string `json:"first_name"`
	LastName              string `json:"last_name"`
	LegalName             string `json:"legal_name"`
	IdentifierCode        string `json:"identifier_code"`
	TaxID                 string `json:"tax_id"`
	PhoneNumber           string `json:"phone_number"`
	Email                 string `json:"email"`
	AddressLine1          string `json:"address_line1"`
	AddressLine2          string `json:"address_line2"`
	Locality              string `json:"locality"`
	County                string `json:"county"`
	Country               string `json:"country"`
	Notes                 string `json:"notes"`
	IsDefaultOrganization bool   `json:"is_default_organization"`
	Active                bool   `json:"active"`
	CreatedAt             string `json:"created_at"`
	UpdatedAt             string `json:"updated_at"`
}

type CreatePartyRequest struct {
	Code                  string `json:"code"`
	InstitutionID         string `json:"institution_id"`
	PartyType             string `json:"party_type"`
	DisplayName           string `json:"display_name"`
	ShortName             string `json:"short_name"`
	FirstName             string `json:"first_name"`
	LastName              string `json:"last_name"`
	LegalName             string `json:"legal_name"`
	IdentifierCode        string `json:"identifier_code"`
	TaxID                 string `json:"tax_id"`
	PhoneNumber           string `json:"phone_number"`
	Email                 string `json:"email"`
	AddressLine1          string `json:"address_line1"`
	AddressLine2          string `json:"address_line2"`
	Locality              string `json:"locality"`
	County                string `json:"county"`
	Country               string `json:"country"`
	Notes                 string `json:"notes"`
	IsDefaultOrganization bool   `json:"is_default_organization"`
	Active                bool   `json:"active"`
}

type UpdatePartyRequest struct {
	Code                  *string `json:"code"`
	PartyType             *string `json:"party_type"`
	DisplayName           *string `json:"display_name"`
	ShortName             *string `json:"short_name"`
	FirstName             *string `json:"first_name"`
	LastName              *string `json:"last_name"`
	LegalName             *string `json:"legal_name"`
	IdentifierCode        *string `json:"identifier_code"`
	TaxID                 *string `json:"tax_id"`
	PhoneNumber           *string `json:"phone_number"`
	Email                 *string `json:"email"`
	AddressLine1          *string `json:"address_line1"`
	AddressLine2          *string `json:"address_line2"`
	Locality              *string `json:"locality"`
	County                *string `json:"county"`
	Country               *string `json:"country"`
	Notes                 *string `json:"notes"`
	IsDefaultOrganization *bool   `json:"is_default_organization"`
	Active                *bool   `json:"active"`
}

type DocumentVersion struct {
	ID              string  `json:"id"`
	DocumentID      string  `json:"document_id"`
	VersionNo       int     `json:"version_no"`
	Subject         string  `json:"subject"`
	DocumentType    string  `json:"document_type"`
	Direction       string  `json:"direction"`
	Status          string  `json:"status"`
	Correspondent   string  `json:"correspondent"`
	AssignedTo      string  `json:"assigned_to"`
	Confidentiality string  `json:"confidentiality"`
	Summary         string  `json:"summary"`
	DueDate         *string `json:"due_date"`
	ChangeNotes     string  `json:"change_notes"`
	CreatedBy       string  `json:"created_by"`
	CreatedAt       string  `json:"created_at"`
}

type DocumentAttachment struct {
	ID         string `json:"id"`
	DocumentID string `json:"document_id"`
	Title      string `json:"title"`
	FileName   string `json:"file_name"`
	MimeType   string `json:"mime_type"`
	StorageKey string `json:"storage_key"`
	SizeBytes  int64  `json:"size_bytes"`
	Category   string `json:"category"`
	Status     string `json:"status"`
	UploadedBy string `json:"uploaded_by"`
	UploadedAt string `json:"uploaded_at"`
}

type DocumentFiltersResponse struct {
	DocumentTypes     []string   `json:"document_types"`
	Directions        []string   `json:"directions"`
	Statuses          []string   `json:"statuses"`
	Confidentialities []string   `json:"confidentialities"`
	Registries        []Registru `json:"registries,omitempty"`
}

type CreateDocumentRequest struct {
	RegistruID      *int64  `json:"registru_id"`
	Subject         string  `json:"subject"`
	DocumentType    string  `json:"document_type"`
	Direction       string  `json:"direction"`
	Status          string  `json:"status"`
	Correspondent   string  `json:"correspondent"`
	AssignedTo      string  `json:"assigned_to"`
	CorrespondentPartyID *string `json:"correspondent_party_id"`
	AssignedPartyID      *string `json:"assigned_party_id"`
	Confidentiality string  `json:"confidentiality"`
	Summary         string  `json:"summary"`
	DueDate         *string `json:"due_date"`
}

type UpdateDocumentRequest struct {
	RegistruID      *int64  `json:"registru_id"`
	Subject         string  `json:"subject"`
	DocumentType    string  `json:"document_type"`
	Direction       string  `json:"direction"`
	Status          string  `json:"status"`
	Correspondent   string  `json:"correspondent"`
	AssignedTo      string  `json:"assigned_to"`
	CorrespondentPartyID *string `json:"correspondent_party_id"`
	AssignedPartyID      *string `json:"assigned_party_id"`
	Confidentiality string  `json:"confidentiality"`
	Summary         string  `json:"summary"`
	DueDate         *string `json:"due_date"`
	ChangeNotes     string  `json:"change_notes"`
}

type CancelDocumentRequest struct {
	Reason string `json:"reason"`
}

type BatchCreateDocumentsRequest struct {
	RegistruID      int64   `json:"registru_id"`
	Count           int     `json:"count"`
	Subject         string  `json:"subject"`
	DocumentType    string  `json:"document_type"`
	Direction       string  `json:"direction"`
	Status          string  `json:"status"`
	Correspondent   string  `json:"correspondent"`
	AssignedTo      string  `json:"assigned_to"`
	CorrespondentPartyID *string `json:"correspondent_party_id"`
	AssignedPartyID      *string `json:"assigned_party_id"`
	Confidentiality string  `json:"confidentiality"`
	Summary         string  `json:"summary"`
	DueDate         *string `json:"due_date"`
}

type ExportDocumentsRequest struct {
	RegistruID *int64  `json:"registru_id"`
	StartDate  *string `json:"start_date"`
	EndDate    *string `json:"end_date"`
}

type CreateRegistruRequest struct {
	Nume         string  `json:"nume"`
	PrefixNr     string  `json:"prefix_nr"`
	NrInceput    int     `json:"nr_inceput"`
	NrCurent     string  `json:"nr_curent"`
	NrUrmator    string  `json:"nr_urmator"`
	DataResetare *string `json:"data_resetare"`
	TipRegistru  string  `json:"tip_registru"`
	IsDefault    bool    `json:"isDefault"`
}

type UpdateRegistruRequest struct {
	Nume         *string `json:"nume"`
	PrefixNr     *string `json:"prefix_nr"`
	NrInceput    *int    `json:"nr_inceput"`
	NrCurent     *string `json:"nr_curent"`
	NrUrmator    *string `json:"nr_urmator"`
	DataResetare *string `json:"data_resetare"`
	TipRegistru  *string `json:"tip_registru"`
	IsDefault    *bool   `json:"isDefault"`
}

type CreateDocumentVersionRequest struct {
	Subject         string  `json:"subject"`
	Status          string  `json:"status"`
	AssignedTo      string  `json:"assigned_to"`
	Confidentiality string  `json:"confidentiality"`
	Summary         string  `json:"summary"`
	DueDate         *string `json:"due_date"`
	ChangeNotes     string  `json:"change_notes"`
}

type CreateDocumentAttachmentRequest struct {
	Title      string `json:"title"`
	FileName   string `json:"file_name"`
	MimeType   string `json:"mime_type"`
	StorageKey string `json:"storage_key"`
	SizeBytes  int64  `json:"size_bytes"`
	Category   string `json:"category"`
	Status     string `json:"status"`
	UploadedBy string `json:"uploaded_by"`
}

type DocumentLookupItem struct {
	ID             string `json:"id"`
	RegistryNumber string `json:"registry_number"`
	Subject        string `json:"subject"`
	DocumentType   string `json:"document_type"`
	Status         string `json:"status"`
}

type LinkedDocument struct {
	LinkID          string `json:"link_id"`
	DocumentID      string `json:"document_id"`
	RegistryNumber  string `json:"registry_number"`
	Subject         string `json:"subject"`
	DocumentType    string `json:"document_type"`
	Status          string `json:"status"`
	RelationType    string `json:"relation_type"`
	RegisteredAt    string `json:"registered_at"`
	Confidentiality string `json:"confidentiality"`
}

type CreateDocumentLinkRequest struct {
	DocumentID     string `json:"document_id"`
	SourceModule   string `json:"source_module"`
	SourceRecordID string `json:"source_record_id"`
	RelationType   string `json:"relation_type"`
}
