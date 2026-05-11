package registratura

type Document struct {
	ID              string  `json:"id"`
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
	DocumentTypes     []string `json:"document_types"`
	Directions        []string `json:"directions"`
	Statuses          []string `json:"statuses"`
	Confidentialities []string `json:"confidentialities"`
}

type CreateDocumentRequest struct {
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
