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
	InstitutionID   string  `json:"institution_id"`
	Confidentiality string  `json:"confidentiality"`
	Summary         string  `json:"summary"`
	RegisteredAt    string  `json:"registered_at"`
	DueDate         *string `json:"due_date"`
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
