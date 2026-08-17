[CmdletBinding()]
param(
    [string]$Router = "backend/cmd/server/main.go",
    [string]$Output = "openapi/openapi.json"
)

$ErrorActionPreference = 'Stop'

function Get-Tag([string]$path) {
    if ($path -like '/api/registratura/*') { return 'Registratura' }
    if ($path -like '/api/workflow/*') { return 'Workflow' }
    if ($path -like '/api/earchiva/*') { return 'eArhiva' }
    if ($path -like '/api/education/*') { return 'Scoala' }
    if ($path -like '/api/admin/*') { return 'Administrare' }
    if ($path -like '/api/gdpr/*') { return 'GDPR' }
    if ($path -like '/api/auth/*' -or $path -eq '/api/me' -or $path -eq '/api/profile') { return 'Authentication' }
    return 'Platform'
}

function Get-OperationId([string]$method, [string]$path) {
    $slug = ($path.Trim('/') -replace '[{}]', '' -replace '[^A-Za-z0-9]+', '_').Trim('_')
    return ("{0}_{1}" -f $method.ToLowerInvariant(), $slug).ToLowerInvariant()
}

function Get-ContractFamily([string]$path) {
    if ($path -like '/api/registratura/*') { return 'Registratura' }
    if ($path -like '/api/workflow/*') { return 'Workflow' }
    if ($path -like '/api/earchiva/*') { return 'Archive' }
    if ($path -like '/api/auth/*' -or $path -like '/api/passkeys/*' -or $path -like '/api/eudi-wallet/*' -or $path -in @('/api/me', '/api/profile')) { return 'Identity' }
    return $null
}

# Router-wide compatibility schemas are forbidden for request bodies.  These
# field sets are the union of the concrete DTO JSON fields in each bounded
# handler family; every generated operation receives its own named, closed
# schema below so an SDK never exposes an `any`/open-object command payload.
function New-ClosedRequestSchema([string]$operationKey) {
	# Reviewed route-to-DTO map. The generator reads JSON tags and Go scalar types
	# from the handler DTO, preventing a field valid for one admin/GDPR/resource
	# command from leaking into another endpoint's SDK type.
	$dtoMap = @{
		'POST /api/passkeys/login-finish'='backend/internal/auth/models.go|FinishPasskeyAuthenticationRequest'; 'POST /api/passkeys/register-finish'='backend/internal/auth/models.go|FinishPasskeyRegistrationRequest'; 'PUT /api/profile'='backend/internal/auth/models.go|UpdateProfileRequest'
		'POST /api/admin/users'='backend/internal/admin/models.go|UpsertUserRequest'; 'POST /api/admin/roles'='backend/internal/admin/models.go|UpsertRoleRequest'; 'POST /api/admin/role-assignments'='backend/internal/admin/models.go|UpsertUserRoleAssignmentRequest'; 'POST /api/admin/role-permissions'='backend/internal/admin/models.go|UpsertRolePermissionAssignmentRequest'; 'POST /api/admin/position-roles'='backend/internal/admin/models.go|UpsertPositionRoleAssignmentRequest'; 'POST /api/admin/org-units'='backend/internal/admin/models.go|UpsertOrgUnitRequest'; 'POST /api/admin/memberships'='backend/internal/admin/models.go|UpsertMembershipRequest'; 'POST /api/admin/positions'='backend/internal/admin/models.go|UpsertPositionRequest'; 'POST /api/admin/permissions/assignments'='backend/internal/admin/models.go|UpsertPermissionAssignmentRequest'; 'POST /api/admin/auth-methods'='backend/internal/admin/models.go|UpdateAuthMethodSettingRequest'; 'POST /api/admin/modules'='backend/internal/admin/models.go|UpdateModuleSettingRequest'; 'POST /api/admin/oidc/clients'='backend/internal/admin/models.go|UpsertOIDCClientRequest'; 'POST /api/admin/gdpr-settings'='backend/internal/admin/models.go|UpdateGdprSettingRequest'; 'POST /api/admin/dossier-requirements'='backend/internal/admin/models.go|CreateDossierRequirementRequest'; 'POST /api/admin/workflow-definitions'='backend/internal/admin/models.go|CreateWorkflowDefinitionRequest'; 'POST /api/admin/nomenclatures'='backend/internal/admin/models.go|CreateNomenclatureRequest'; 'POST /api/admin/education-taxonomies'='backend/internal/admin/models.go|CreateEducationTaxonomyRequest'
		'PATCH /api/registratura/documents/{documentID}'='backend/internal/registratura/models.go|UpdateDocumentRequest'; 'POST /api/registratura/documents/batch'='backend/internal/registratura/models.go|BatchCreateDocumentsRequest'; 'POST /api/registratura/documents/export-pdf'='backend/internal/registratura/models.go|ExportDocumentsRequest'; 'POST /api/registratura/documents/{documentID}/versions'='backend/internal/registratura/models.go|CreateDocumentVersionRequest'; 'POST /api/registratura/documents/{documentID}/attachments'='backend/internal/registratura/models.go|CreateDocumentAttachmentRequest'; 'POST /api/registratura/registre'='backend/internal/registratura/models.go|CreateRegistruRequest'; 'PATCH /api/registratura/registre/{id}'='backend/internal/registratura/models.go|UpdateRegistruRequest'; 'POST /api/registratura/parties'='backend/internal/registratura/models.go|CreatePartyRequest'; 'PATCH /api/registratura/parties/{id}'='backend/internal/registratura/models.go|UpdatePartyRequest'; 'POST /api/registratura/admin/departments'='backend/internal/registratura/structure.go|departmentRequest'; 'PATCH /api/registratura/admin/departments/{id}'='backend/internal/registratura/structure.go|departmentRequest'; 'POST /api/registratura/admin/organizations'='backend/internal/registratura/structure.go|organizationRequest'; 'PATCH /api/registratura/admin/organizations/{id}'='backend/internal/registratura/structure.go|organizationRequest'; 'PUT /api/registratura/admin/users/{id}/assignments'='backend/internal/registratura/structure.go|assignmentRequest'; 'POST /api/registratura/admin/registries'='backend/internal/registratura/structure.go|adminRegistryRequest'; 'PATCH /api/registratura/admin/registries/{id}'='backend/internal/registratura/structure.go|adminRegistryRequest'; 'POST /api/registratura/document-links'='backend/internal/registratura/models.go|CreateDocumentLinkRequest'
		'POST /api/gdpr/retention-policies'='backend/internal/gdpr/models.go|CreateRetentionPolicyRequest'; 'POST /api/gdpr/subject-requests'='backend/internal/gdpr/models.go|CreateSubjectRequestRequest'; 'POST /api/gdpr/exports'='backend/internal/gdpr/models.go|CreateSubjectExportRequest'; 'POST /api/gdpr/publication-reviews'='backend/internal/gdpr/models.go|CreatePublicationReviewRequest'
		'POST /api/earchiva/classification-reviews/{reviewID}/approve'='backend/internal/earchiva/archive_classification.go|ArchiveClassificationApprovalRequest'; 'POST /api/earchiva/classification-reviews/{reviewID}/correct'='backend/internal/earchiva/archive_classification.go|ArchiveClassificationCorrectionRequest'
	}
	if ($dtoMap.ContainsKey($operationKey)) {
		$parts=$dtoMap[$operationKey].Split('|'); $previousGo111Module=$env:GO111MODULE; $env:GO111MODULE='off'; $json=& go run ./scripts/openapi/go-schema-helper -- $parts[0] $parts[1]; $env:GO111MODULE=$previousGo111Module
		if($LASTEXITCODE -ne 0){throw "DTO schema helper failed for ${operationKey}"}; return ($json -join "`n" | ConvertFrom-Json -AsHashtable)
	}
	if ($operationKey -eq 'POST /api/earchiva/documents') {
		# The archive upload handler consumes a multipart PDF plus scalar metadata.
		# The file is the only required part; title/source/taxonomy values are
		# optional because the handler derives safe defaults when omitted.
		$binary = [ordered]@{ type = 'string'; format = 'binary' }
		$string = [ordered]@{ type = 'string' }
		$metadata = [ordered]@{ type = 'string'; contentMediaType = 'application/json'; description = 'JSON object encoded as one multipart text part.' }
		return [ordered]@{
			type = 'object'; additionalProperties = $false; required = @('file')
			properties = [ordered]@{ file = $binary; title = $string; source_kind = $string; source_system = $string; external_reference = $string; taxonomy_code = $string; taxonomy_label = $string; taxonomy_parent_code = $string; document_date = $string; metadata = $metadata; idempotency_key = $string }
		}
	}
    $string = [ordered]@{ type = 'string' }
    $boolean = [ordered]@{ type = 'boolean' }
    $integer = [ordered]@{ type = 'integer' }
    $arrayString = [ordered]@{ type = 'array'; items = [ordered]@{ type = 'string' } }
    $props = [ordered]@{}
    if ($operationKey -match '^POST /api/passkeys/login-options') { $props['user_verification'] = $string }
    elseif ($operationKey -match '^POST /api/passkeys/login-finish') { $props['challenge']=$string; $props['credential_id']=$string; $props['response']=[ordered]@{type='object';additionalProperties=$true;'x-free-form-property'=$true} }
    elseif ($operationKey -match '^POST /api/passkeys/register-options') { $props['device_name']=$string }
    elseif ($operationKey -match '^POST /api/passkeys/register-finish') { $props['credential_id']=$string; $props['device_name']=$string; $props['challenge']=$string; $props['response']=[ordered]@{type='object';additionalProperties=$true;'x-free-form-property'=$true} }
    elseif ($operationKey -match '^POST /api/eudi-wallet/activate') { }
    elseif ($operationKey -match '^PUT /api/profile') { $props['name']=$string; $props['phone_number']=$string; $props['locale']=[ordered]@{type='string';enum=@('ro','en')} }
    elseif ($operationKey -match '^POST /api/admin/') {
        foreach($field in @('id','name','email','phone','locale','status','preferred_otp_channel','code','label','user_id','role_code','permission_code','position_code','org_unit_code','organization_name','start_date','end_date','scope_module','source_module','relation_type','domain','label_ro','label_en','category','initial_step','client_id','client_name','value_type','value_text','parent_code')) { $props[$field]=$string }
        foreach($field in @('email_verified','phone_verified','assigned','is_primary','active','required_for_readiness','required_for_submit','required_for_approve','enabled','primary_method','public_client','require_pkce','value_bool')) { $props[$field]=$boolean }
        foreach($field in @('sort_order','min_count','sla_hours','value_int')) { $props[$field]=$integer }
        $props['redirect_uris']=$arrayString
    }
    elseif ($operationKey -match '^POST /api/gdpr/') {
        foreach($field in @('id','title','description','subject_id','subject_email','request_type','status','due_date','retention_code','legal_basis','format','notes','document_id','reviewer_id','decision','publication_url')) { $props[$field]=$string }
        foreach($field in @('active','contains_personal_data','approved')) { $props[$field]=$boolean }
        $props['metadata']=[ordered]@{type='object';additionalProperties=$true;'x-free-form-property'=$true}
    }
    elseif ($operationKey -match '^/api/registratura/' -or $operationKey -match ' /api/registratura/') {
        foreach($field in @('subject','document_type','direction','status','correspondent','assigned_to','confidentiality','summary','due_date','change_notes','title','file_name','mime_type','storage_key','category','uploaded_by','nume','prefix_nr','nr_curent','nr_urmator','data_resetare','tip_registru','name','description','parent_id','role_tag','prefix','current_number','next_number','registry_type','primary_department_id','organization_id','document_id','source_module','source_record_id','relation_type','action','note')) { $props[$field]=$string }
        foreach($field in @('isDefault','is_default','active','assigned')) { $props[$field]=$boolean }
        foreach($field in @('registru_id','count','nr_inceput','start_number','size_bytes','expected_version')) { $props[$field]=$integer }
        foreach($field in @('department_ids')) { $props[$field]=$arrayString }
        $props['correspondent_party_id']=$string; $props['assigned_party_id']=$string; $props['department_id']=$string; $props['user_id']=$string
    }
    elseif ($operationKey -match '^POST /api/earchiva/documents') { $props['title']=$string; $props['classificationCode']=$string; $props['recordId']=[ordered]@{type='string';format='uuid'}; $props['metadata']=[ordered]@{type='object';additionalProperties=$true;'x-free-form-property'=$true} }
    else { $props['format']=$string; $props['record_id']=$string }
    return [ordered]@{type='object';additionalProperties=$false;properties=$props}
}

function Get-GoDTOObjectSchema([string]$sourceDTO) {
    if(-not $script:goSchemaHelper){$previousGo111Module=$env:GO111MODULE; $env:GO111MODULE='off'; & go build -o scripts/openapi/go-schema-helper/openapi-schema-helper.exe ./scripts/openapi/go-schema-helper; $env:GO111MODULE=$previousGo111Module; if($LASTEXITCODE -ne 0){throw 'could not build Go DTO schema helper'}; $script:goSchemaHelper=(Resolve-Path scripts/openapi/go-schema-helper/openapi-schema-helper.exe)}
    $parts=$sourceDTO.Split('|'); $json=& $script:goSchemaHelper $parts[0] $parts[1]
    if($LASTEXITCODE -ne 0){throw "DTO schema helper failed for $sourceDTO"}; return ($json -join "`n" | ConvertFrom-Json -AsHashtable)
}

function Get-ResponseDTO([string]$path) {
    $map=@{
      '/api/admin/users'='backend/internal/admin/models.go|AdminUser'; '/api/admin/roles'='backend/internal/admin/models.go|Role'; '/api/admin/role-assignments'='backend/internal/admin/models.go|UserRoleAssignment'; '/api/admin/role-permissions'='backend/internal/admin/models.go|RolePermissionAssignment'; '/api/admin/position-roles'='backend/internal/admin/models.go|PositionRoleAssignment'; '/api/admin/org-units'='backend/internal/admin/models.go|OrgUnit'; '/api/admin/memberships'='backend/internal/admin/models.go|Membership'; '/api/admin/positions'='backend/internal/admin/models.go|Position'; '/api/admin/permissions'='backend/internal/admin/models.go|Permission'; '/api/admin/permissions/assignments'='backend/internal/admin/models.go|PermissionAssignment'; '/api/admin/auth-methods'='backend/internal/admin/models.go|AuthMethodSetting'; '/api/admin/modules'='backend/internal/admin/models.go|ModuleSetting'; '/api/admin/oidc/clients'='backend/internal/admin/models.go|OIDCClient'; '/api/admin/gdpr-settings'='backend/internal/admin/models.go|GdprSetting'; '/api/admin/dossier-requirements'='backend/internal/admin/models.go|DossierRequirement'; '/api/admin/workflow-definitions'='backend/internal/admin/models.go|WorkflowDefinition'; '/api/admin/nomenclatures'='backend/internal/admin/models.go|Nomenclature'; '/api/admin/education-taxonomies'='backend/internal/admin/models.go|EducationTaxonomy'
      '/api/gdpr/retention-policies'='backend/internal/gdpr/models.go|RetentionPolicy'; '/api/gdpr/subject-requests'='backend/internal/gdpr/models.go|SubjectRequest'; '/api/gdpr/exports'='backend/internal/gdpr/models.go|SubjectExport'; '/api/gdpr/publication-reviews'='backend/internal/gdpr/models.go|PublicationReview'
    }
    return $map[$path]
}

# Response DTOs are deliberately mapped per operation.  The prior family-level
# fallbacks made generated clients claim that unrelated endpoints returned the
# same shape (for example an archive taxonomy and an archive dashboard).  Keep
# the representation next to the generator so route registration remains the
# source of truth and every success response is closed and useful to an SDK.
function Get-ExactResponseSpec([string]$operationKey) {
    $map = @{
        'GET /api/auth/methods'='object|auth_methods'; 'GET /api/auth/ui-config'='object|auth_ui_config'; 'GET /api/auth/role-catalog'='dto|backend/internal/auth/models.go|RoleCatalogResponse'; 'GET /api/auth/role-positions'='dto|backend/internal/auth/models.go|RolePositionResponse'
        'POST /api/passkeys/login-options'='object|passkey_login_options'; 'POST /api/passkeys/login-finish'='object|passkey_login_finish'; 'POST /api/passkeys/register-options'='object|passkey_register_options'; 'POST /api/passkeys/register-finish'='dto|backend/internal/auth/models.go|PasskeyCredentialSummary'; 'POST /api/eudi-wallet/activate'='object|eudi_activation'
        'GET /api/passkeys'='array|backend/internal/auth/models.go|PasskeyCredentialSummary'
        'GET /api/registratura/documents/filters'='dto|backend/internal/registratura/models.go|DocumentFiltersResponse'; 'GET /api/registratura/nomenclatures'='dto|backend/internal/registratura/models.go|DocumentFiltersResponse'; 'PATCH /api/registratura/documents/{documentID}'='dto|backend/internal/registratura/models.go|Document'; 'GET /api/registratura/documents/{documentID}'='dto|backend/internal/registratura/models.go|Document'; 'POST /api/registratura/documents/{documentID}/cancel'='dto|backend/internal/registratura/models.go|Document'; 'GET /api/registratura/documents/lookup'='array|backend/internal/registratura/models.go|DocumentLookupItem'; 'GET /api/registratura/documents/{documentID}/versions'='array|backend/internal/registratura/models.go|DocumentVersion'; 'POST /api/registratura/documents/{documentID}/versions'='dto|backend/internal/registratura/models.go|DocumentVersion'; 'GET /api/registratura/documents/{documentID}/workflow-history'='array|backend/internal/registratura/models.go|DocumentWorkflowEvent'; 'GET /api/registratura/workflow-assignees'='object|workflow_assignees'; 'GET /api/registratura/documents/{documentID}/attachments'='array|backend/internal/registratura/models.go|DocumentAttachment'; 'POST /api/registratura/documents/{documentID}/attachments'='dto|backend/internal/registratura/models.go|DocumentAttachment'
        'POST /api/registratura/registre'='dto|backend/internal/registratura/models.go|Registru'; 'GET /api/registratura/registre/{id}'='dto|backend/internal/registratura/models.go|Registru'; 'PATCH /api/registratura/registre/{id}'='dto|backend/internal/registratura/models.go|Registru'; 'DELETE /api/registratura/registre/{id}'='empty|'; 'PATCH /api/registratura/registre/{id}/set-default'='dto|backend/internal/registratura/models.go|Registru'
        'GET /api/registratura/parties'='page|backend/internal/registratura/models.go|Party'; 'POST /api/registratura/parties'='dto|backend/internal/registratura/models.go|Party'; 'GET /api/registratura/parties/lookup'='array|backend/internal/registratura/models.go|Party'; 'GET /api/registratura/parties/default-organization'='dto|backend/internal/registratura/models.go|Party'; 'GET /api/registratura/parties/{id}'='dto|backend/internal/registratura/models.go|Party'; 'PATCH /api/registratura/parties/{id}'='dto|backend/internal/registratura/models.go|Party'; 'DELETE /api/registratura/parties/{id}'='empty|'
        'GET /api/registratura/admin/departments'='page|backend/internal/registratura/models.go|Department'; 'POST /api/registratura/admin/departments'='dto|backend/internal/registratura/models.go|Department'; 'PATCH /api/registratura/admin/departments/{id}'='dto|backend/internal/registratura/models.go|Department'; 'DELETE /api/registratura/admin/departments/{id}'='empty|'; 'GET /api/registratura/admin/organizations'='array|backend/internal/registratura/models.go|Organization'; 'POST /api/registratura/admin/organizations'='dto|backend/internal/registratura/models.go|Organization'; 'PATCH /api/registratura/admin/organizations/{id}'='dto|backend/internal/registratura/models.go|Organization'; 'DELETE /api/registratura/admin/organizations/{id}'='empty|'; 'GET /api/registratura/admin/organization-chart'='arrayinline|organization_chart'; 'GET /api/registratura/admin/users/{id}/assignments'='object|user_assignments'; 'PUT /api/registratura/admin/users/{id}/assignments'='object|user_assignments'; 'GET /api/registratura/admin/registries'='page|backend/internal/registratura/structure.go|adminRegistry'; 'POST /api/registratura/admin/registries'='dto|backend/internal/registratura/structure.go|adminRegistry'; 'PATCH /api/registratura/admin/registries/{id}'='dto|backend/internal/registratura/structure.go|adminRegistry'; 'DELETE /api/registratura/admin/registries/{id}'='empty|'; 'GET /api/registratura/document-links'='array|backend/internal/registratura/models.go|LinkedDocument'; 'POST /api/registratura/document-links'='dto|backend/internal/registratura/models.go|LinkedDocument'; 'DELETE /api/registratura/document-links/{linkID}'='empty|'
        'GET /api/workflow/dashboard'='object|workflow_dashboard'; 'GET /api/workflow/definitions'='array|backend/internal/workflow/models.go|Definition'; 'POST /api/workflow/tasks'='dto|backend/internal/workflow/models.go|Task'; 'GET /api/workflow/tasks/filters'='dto|backend/internal/workflow/models.go|FiltersResponse'
		'GET /api/earchiva/dashboard'='object|archive_dashboard'; 'GET /api/earchiva/records/filters'='dto|backend/internal/earchiva/models.go|FiltersResponse'; 'GET /api/earchiva/nomenclatures'='dto|backend/internal/earchiva/models.go|FiltersResponse'; 'GET /api/earchiva/documents/{documentID}'='dto|backend/internal/earchiva/archive_documents.go|ArchiveDocumentDetail'; 'GET /api/earchiva/documents/{documentID}/versions'='array|backend/internal/earchiva/archive_documents.go|ArchiveDocumentVersionSummary'; 'GET /api/earchiva/taxonomy'='array|backend/internal/earchiva/archive_documents.go|ArchiveTaxonomyNode'; 'GET /api/earchiva/admin/health'='dto|backend/internal/earchiva/archive_admin.go|ArchiveAdminHealth'; 'GET /api/earchiva/admin/stats'='dto|backend/internal/earchiva/archive_admin.go|ArchiveAdminStats'; 'GET /api/earchiva/admin/jobs'='dto|backend/internal/earchiva/archive_admin.go|ArchiveAdminJobPage'; 'POST /api/earchiva/admin/jobs/{jobID}/retry'='dto|backend/internal/earchiva/archive_admin.go|ArchiveAdminJob'
		'GET /api/earchiva/classification-reviews'='dto|backend/internal/earchiva/archive_classification.go|ArchiveClassificationReviewPage'; 'POST /api/earchiva/classification-reviews/{reviewID}/approve'='dto|backend/internal/earchiva/archive_classification.go|ArchiveClassificationReview'; 'POST /api/earchiva/classification-reviews/{reviewID}/correct'='dto|backend/internal/earchiva/archive_classification.go|ArchiveClassificationReview'
        'GET /api/registratura/documents'='page|backend/internal/registratura/models.go|Document'; 'GET /api/registratura/registre'='page|backend/internal/registratura/models.go|Registru'; 'GET /api/workflow/tasks'='page|backend/internal/workflow/models.go|Task'; 'GET /api/earchiva/records'='page|backend/internal/earchiva/models.go|Record'; 'GET /api/earchiva/documents'='page|backend/internal/earchiva/archive_documents.go|ArchiveDocumentSearchResult'
        'GET /api/admin/dashboard'='object|admin_dashboard'; 'GET /api/admin/users/filters'='object|admin_user_filters'; 'GET /api/admin/role-permissions/filters'='object|admin_role_permission_filters'; 'GET /api/admin/position-roles/filters'='object|admin_position_role_filters'; 'GET /api/admin/permissions/assignments/filters'='object|admin_permission_assignment_filters'; 'GET /api/admin/audit'='page|backend/internal/admin/models.go|AuditEvent'; 'GET /api/admin/audit/filters'='object|admin_audit_filters'; 'GET /api/admin/dossier-requirements/filters'='object|admin_dossier_filters'; 'GET /api/admin/workflow-definitions/filters'='object|admin_workflow_filters'; 'GET /api/admin/nomenclatures/filters'='object|admin_nomenclature_filters'; 'GET /api/admin/education-taxonomies/filters'='object|admin_education_taxonomy_filters'
        'GET /api/gdpr/dashboard'='dto|backend/internal/gdpr/models.go|DashboardResponse'; 'GET /api/gdpr/config'='dto|backend/internal/gdpr/models.go|ConfigResponse'; 'GET /api/gdpr/retention-policies/filters'='dto|backend/internal/gdpr/models.go|RetentionPolicyFiltersResponse'; 'GET /api/gdpr/subject-requests/filters'='dto|backend/internal/gdpr/models.go|SubjectRequestFiltersResponse'; 'GET /api/gdpr/exports/dashboard'='dto|backend/internal/gdpr/models.go|ExportDashboardResponse'; 'GET /api/gdpr/exports/filters'='dto|backend/internal/gdpr/models.go|SubjectExportFiltersResponse'; 'GET /api/gdpr/publication-reviews/dashboard'='dto|backend/internal/gdpr/models.go|PublicationDashboardResponse'; 'GET /api/gdpr/publication-reviews/filters'='dto|backend/internal/gdpr/models.go|PublicationReviewFiltersResponse'
    }
    if (-not $map.ContainsKey($operationKey)) { return $null }
    return $map[$operationKey].Split('|', 3)
}

function New-ExactInlineResponseSchema([string]$name) {
    $string=@{type='string'}; $boolean=@{type='boolean'}; $integer=@{type='integer'}; $strings=@{type='array';items=@{type='string'}}
    switch ($name) {
        'auth_methods' { return @{type='object';additionalProperties=$false;required=@('methods');properties=@{methods=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{code=$string;label=$string;enabled=$boolean;primary=$boolean}}}}} }
        'auth_ui_config' { return @{type='object';additionalProperties=$false;properties=@{auth_flow=$string;default_locale=$string;available_locales=$strings;theme_family=$string;theme_brand=$string;oidc_issuer=@{type='string';format='uri'};oidc_client_id=$string;desktop_client_id=$string;sms_otp_enabled=$boolean;passkey_enabled=$boolean;eudi_wallet_enabled=$boolean;gdpr_features_enabled=$boolean}} }
        'passkey_login_options' { return @{type='object';additionalProperties=$false;required=@('status','options');properties=@{status=$string;options=@{type='object';additionalProperties=$false;properties=@{challenge=$string;rp=@{type='object';additionalProperties=$false;properties=@{name=$string;id=$string}};timeout=$integer;userVerification=$string;allowCredentials=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{type=$string;id=$string}}}}}}} }
        'passkey_login_finish' { return @{type='object';additionalProperties=$false;required=@('nonce');properties=@{nonce=$string}} }
        'passkey_register_options' { return @{type='object';additionalProperties=$false;properties=@{challenge=$string;rp=@{type='object';additionalProperties=$false;properties=@{name=$string;id=$string}};user=@{type='object';additionalProperties=$false;properties=@{id=$string;name=$string;displayName=$string}};pubKeyCredParams=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{type=$string;alg=$integer}}};timeout=$integer;attestation=$string}} }
        'eudi_activation' { return @{type='object';additionalProperties=$false;required=@('status');properties=@{status=@{type='string';enum=@('active')}}} }
        'organization_chart' { return @{type='object';additionalProperties=$false;properties=@{id=$string;name=$string;description=$string;parent_id=@{type=@('string','null')};role_tag=$string;user_count=$integer;users=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{id=$string;name=$string;email=$string}}};children=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{id=$string;name=$string;description=$string;parent_id=@{type=@('string','null')};role_tag=$string;user_count=$integer}}}}} }
        'user_assignments' { return @{type='object';additionalProperties=$false;properties=@{user_id=$string;department_ids=$strings;primary_department_id=@{type=@('string','null')};organization_id=@{type=@('string','null')}}} }
        'workflow_assignees' { return @{type='object';additionalProperties=$false;properties=@{users=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{id=$string;name=$string;email=$string}}};departments=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{id=$string;name=$string}}}}} }
        'workflow_dashboard' { return @{type='object';additionalProperties=$false;required=@('stats');properties=@{stats=@{type='object';additionalProperties=$false;properties=@{active_tasks=$integer;overdue_tasks=$integer;waiting_approval=$integer;active_definitions=$integer;ready_dossiers=$integer;blocked_dossiers=$integer}}}} }
        'archive_dashboard' { return @{type='object';additionalProperties=$false;required=@('stats');properties=@{stats=@{type='object';additionalProperties=$false;properties=@{total_records=$integer;validated_records=$integer;draft_records=$integer;unique_fonds=$integer}}}} }
        'admin_dashboard' { return @{type='object';additionalProperties=$false;properties=@{stats=@{type='object';additionalProperties=$false;properties=@{users=$integer;memberships=$integer;positions=$integer;permissions=$integer;workflows=$integer;archives=$integer;ready_dossiers=$integer;blocked_dossiers=$integer}};modules=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{code=$string;active=$boolean}}};admin_sections=$strings;warnings=$strings}} }
        'admin_user_filters' { return @{type='object';additionalProperties=$false;properties=@{positions=$strings;statuses=$strings;locales=$strings}} }
        'admin_role_permission_filters' { return @{type='object';additionalProperties=$false;properties=@{roles=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{code=$string;label=$string}}};permissions=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{code=$string;label=$string}}}}} }
        'admin_position_role_filters' { return @{type='object';additionalProperties=$false;properties=@{positions=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{code=$string;name=$string}}};roles=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{code=$string;label=$string}}}}} }
        'admin_permission_assignment_filters' { return @{type='object';additionalProperties=$false;properties=@{permissions=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{code=$string;label=$string}}};positions=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{code=$string;name=$string}}}}} }
        'admin_audit_filters' { return @{type='object';additionalProperties=$false;properties=@{domains=$strings;target_types=$strings;statuses=$strings}} }
        'admin_dossier_filters' { return @{type='object';additionalProperties=$false;properties=@{source_modules=$strings;relation_types=$strings}} }
        'admin_workflow_filters' { return @{type='object';additionalProperties=$false;properties=@{categories=$strings}} }
        'admin_nomenclature_filters' { return @{type='object';additionalProperties=$false;properties=@{domains=$strings}} }
        'admin_education_taxonomy_filters' { return @{type='object';additionalProperties=$false;properties=@{domains=$strings}} }
        'gdpr_dashboard' { return @{type='object';additionalProperties=$false;properties=@{stats=@{type='object';additionalProperties=$false;properties=@{}}}} }
        'gdpr_config' { return @{type='object';additionalProperties=$false;properties=@{enabled=$boolean;retention_enabled=$boolean;subject_requests_enabled=$boolean;exports_enabled=$boolean;publication_reviews_enabled=$boolean}} }
        'gdpr_retention_filters' { return @{type='object';additionalProperties=$false;properties=@{statuses=$strings;legal_bases=$strings}} }
        'gdpr_subject_request_filters' { return @{type='object';additionalProperties=$false;properties=@{request_types=$strings;statuses=$strings}} }
        'gdpr_export_dashboard' { return @{type='object';additionalProperties=$false;properties=@{stats=@{type='object';additionalProperties=$false;properties=@{}}}} }
        'gdpr_export_filters' { return @{type='object';additionalProperties=$false;properties=@{formats=$strings;statuses=$strings}} }
        'gdpr_publication_dashboard' { return @{type='object';additionalProperties=$false;properties=@{stats=@{type='object';additionalProperties=$false;properties=@{}}}} }
        'gdpr_publication_filters' { return @{type='object';additionalProperties=$false;properties=@{statuses=$strings;decisions=$strings}} }
        default { throw "Unknown exact inline response '$name'" }
    }
}

function Add-ExactResponseSchema([hashtable]$schemas, [string]$operationId, [string[]]$spec) {
    $responseName="${operationId}_response"; if($schemas.Contains($responseName)){ return $responseName }
    $kind=$spec[0]; $source=if($spec.Count -gt 1){($spec[1..($spec.Count-1)] -join '|')}else{''}
    if($kind -eq 'empty'){ return $null }
    if($kind -eq 'object') { $schemas[$responseName]=New-ExactInlineResponseSchema $source; return $responseName }
    if($kind -eq 'arrayinline') { $schemas[$responseName]=@{type='array';items=(New-ExactInlineResponseSchema $source);additionalProperties=$false}; return $responseName }
    $parts=$source.Split('|',2); $itemName="${operationId}_item"; if(-not $schemas.Contains($itemName)){$schemas[$itemName]=Get-GoDTOObjectSchema $source}
    if($kind -eq 'dto'){$schemas[$responseName]=$schemas[$itemName]; return $responseName}
    if($kind -eq 'array'){$schemas[$responseName]=@{type='array';items=@{'$ref'="#/components/schemas/$itemName"};additionalProperties=$false}; return $responseName}
    if($kind -eq 'page'){$schemas[$responseName]=@{type='object';additionalProperties=$false;required=@('items','page','pageSize','total');properties=@{items=@{type='array';items=@{'$ref'="#/components/schemas/$itemName"}};page=@{type='integer';minimum=1};pageSize=@{type='integer';minimum=1};total=@{type='integer';minimum=0}}}; return $responseName}
    throw "Unknown exact response kind '$kind'"
}

$routerSource = Get-Content -Raw $Router
$common = Get-Content -Raw 'openapi/components/common.json' | ConvertFrom-Json -AsHashtable
$overrides = Get-Content -Raw 'openapi/overrides.json' | ConvertFrom-Json -AsHashtable
$domainRules = @()
$domainCoverage = @{}

# Domain fragments are deliberately kept separate so individual backend areas can be
# audited without creating merge conflicts. Generation is the single deterministic
# composition point; duplicate route overrides or schemas fail the build.
Get-ChildItem 'openapi/domains/*.overrides.json' | Sort-Object Name | ForEach-Object {
    $fragment = Get-Content -Raw $_.FullName | ConvertFrom-Json -AsHashtable
    if ($fragment.Contains('rules')) {
        $domainRules += @($fragment.rules)
        if ($fragment.Contains('schemas')) {
            foreach ($schemaName in $fragment.schemas.Keys) {
                if ($common.components.schemas.Contains($schemaName)) { throw "Duplicate OpenAPI schema '$schemaName' in $($_.Name)" }
                $common.components.schemas[$schemaName] = $fragment.schemas[$schemaName]
            }
        }
        return
    }

    foreach ($operationKey in $fragment.Keys) {
        if ($operationKey -eq '$schema') { continue }
        if ($overrides.Contains($operationKey)) { throw "Duplicate OpenAPI operation override '$operationKey' in $($_.Name)" }
        $overrides[$operationKey] = $fragment[$operationKey]
    }
}

Get-ChildItem 'openapi/domains/*.coverage.json' | Sort-Object Name | ForEach-Object {
    $coverage = Get-Content -Raw $_.FullName | ConvertFrom-Json -AsHashtable
    if ($coverage.Contains('components') -and $coverage.components.Contains('schemas')) {
        foreach ($schemaName in $coverage.components.schemas.Keys) {
            if ($common.components.schemas.Contains($schemaName)) { throw "Duplicate OpenAPI schema '$schemaName' in $($_.Name)" }
            $common.components.schemas[$schemaName] = $coverage.components.schemas[$schemaName]
        }
    }
    if ($coverage.Contains('operations')) {
        foreach ($coveredOperation in $coverage.operations) {
            if ($domainCoverage.Contains($coveredOperation.operationKey)) { throw "Duplicate domain coverage '$($coveredOperation.operationKey)'" }
            $domainCoverage[$coveredOperation.operationKey] = $coveredOperation
        }
    }
}
# These compatibility envelopes must not leak into any generated operation.
# Per-operation response schemas are installed below from handler DTOs.
foreach($compatibilitySchema in @('IdentityResponse','RegistraturaResponse','WorkflowResponse','ArchiveResponse','AdminListResponse','GdprListResponse','Page')) { [void]$common.components.schemas.Remove($compatibilitySchema) }
$routePattern = [regex]'\.(Get|Post|Put|Patch|Delete)\("([^"]+)"'
$publicPaths = @('/api/config', '/api/meta/app', '/api/auth/methods', '/api/auth/ui-config', '/api/auth/role-catalog', '/api/auth/role-positions')
$paths = [ordered]@{}
$operations = @()

foreach ($match in $routePattern.Matches($routerSource)) {
    $method = $match.Groups[1].Value.ToLowerInvariant()
    $registeredPath = $match.Groups[2].Value
    $path = if ($registeredPath -in @('/health', '/readyz', '/healthz')) { $registeredPath } elseif ($registeredPath -eq '/logout') { '/logout' } else { "/api$registeredPath" }
    $key = "{0} {1}" -f $method.ToUpperInvariant(), $path
    $lineStart = $routerSource.LastIndexOf("`n", $match.Index) + 1
    $lineEnd = $routerSource.IndexOf("`n", $match.Index)
    if ($lineEnd -lt 0) { $lineEnd = $routerSource.Length }
    $line = $routerSource.Substring($lineStart, $lineEnd - $lineStart)
    $permission = $null
    $contextStart = [Math]::Max(0, $match.Index - 2000)
    $routeContext = $routerSource.Substring($contextStart, $match.Index - $contextStart)
    $permissionMatches = [regex]::Matches($routeContext, 'RequirePermissions\("([^"]+)"\)')
    if ($permissionMatches.Count -gt 0) { $permission = $permissionMatches[$permissionMatches.Count - 1].Groups[1].Value }
    elseif ($routeContext -match 'RequireAnyPermissions\(') { $permission = 'one of the permissions required by this route' }

    $isPublic = $path -in $publicPaths -or $path -in @('/health', '/readyz', '/healthz', '/logout')
    $override = $overrides[$key]
    $coverage = $domainCoverage[$key]
    $matchingRules = @($domainRules | Where-Object { $path -match $_.match })
    if ($matchingRules.Count -gt 1) { throw "Multiple domain rules match '$key'" }
    $domainRule = if ($matchingRules.Count -eq 1) { $matchingRules[0] } else { $null }

    if (-not $override -and $domainRule) {
        $responseSchemaFromRule = if ($method -eq 'get') { $domainRule.getResponse } else { $domainRule.writeResponse }
        $override = [ordered]@{
            summary = "$($method.ToUpperInvariant()) $path"
            description = "Reviewed $($domainRule.domain ?? 'domain') route contract. Backend authorization and tenant scoping remain authoritative."
            tags = @($domainRule.tag)
            status = $domainRule.status
            responseSchema = $responseSchemaFromRule
            requestSchema = $domainRule.writeRequest
            tenantScope = $domainRule.tenantScope
            security = $domainRule.security
            queryParameters = @($domainRule.query)
            errors = @($domainRule.errors)
            responseContentType = $domainRule.contentType
            requestContentType = $domainRule.contentType
        }
    }

    if ($coverage) {
        $override.requiredPermission = $coverage.requiredPermission
        $override.pathParameters = @($coverage.pathParameters)
        $override.queryParameters = @($coverage.queryParameters)
        $override.errors = @($coverage.errors)
        $override.responseStatus = [string]$coverage.success.status
        $override.responseContentType = [string]$coverage.success.contentType
        # Coverage is handler-backed.  Carry its concrete request/response model
        # into the generated operation rather than falling back to a family-wide
        # compatibility schema.
        if ($coverage.success.schema) {
            $coverageResponseSchema = [string]$coverage.success.schema
            # Handler catalogues use the OpenAPI primitive marker `binary` for
            # streamed exports.  The composed document uses named schemas so
            # every reference remains resolvable and its media type is explicit.
            if ($coverageResponseSchema -eq 'binary') {
                $coverageResponseSchema = if ([string]$coverage.success.contentType -eq 'text/csv') { 'BinaryCsv' } else { 'BinaryPdf' }
            }
            $override.responseSchema = $coverageResponseSchema
        }
        if ($coverage.requestBody -and $coverage.requestBody.schema) { $override.requestSchema = [string]$coverage.requestBody.schema }
        if ($coverage.requestBody) { $override.requestContentType = [string]$coverage.requestBody.contentType }
    }
    [object[]]$tag = if ($override -and $override.tags) { @($override.tags) } else { @((Get-Tag $path)) }
    $family = Get-ContractFamily $path
    $isDetailedFamily = $null -ne $family
    $responseSchema = if ($override -and $override.responseSchema) { [string]$override.responseSchema } elseif ($isDetailedFamily) { "$family`Response" } else { 'Entity' }
    $operationId = Get-OperationId $method $path
    $exactResponse = Get-ExactResponseSpec $key
    $exactEmptyResponse = $false
    if ($exactResponse) {
        if ($exactResponse[0] -eq 'empty') { $exactEmptyResponse=$true; $responseSchema=$null }
        else { $responseSchema=Add-ExactResponseSchema $common.components.schemas $operationId $exactResponse }
    }
    $responseDTO=Get-ResponseDTO $path
    if($responseDTO -and $responseSchema -in @('AdminListResponse','GdprListResponse')) {
        $operationSlug=Get-OperationId $method $path; $itemName="${operationSlug}_item"; $responseSchema="${operationSlug}_response"
        if(-not $common.components.schemas.Contains($itemName)){$common.components.schemas[$itemName]=Get-GoDTOObjectSchema $responseDTO}
        if(-not $common.components.schemas.Contains($responseSchema)){$common.components.schemas[$responseSchema]=[ordered]@{type='object';required=@('items');additionalProperties=$false;properties=[ordered]@{items=[ordered]@{type='array';items=[ordered]@{'$ref'="#/components/schemas/$itemName"}};page=[ordered]@{type='integer';minimum=1};pageSize=[ordered]@{type='integer';minimum=1};total=[ordered]@{type='integer';minimum=0}}}}
    }
    $operation = [ordered]@{
        operationId = $operationId
        summary = if ($override -and $override.summary) { [string]$override.summary } else { "$(($method.ToUpperInvariant())) $path" }
        description = if ($override -and $override.description) { [string]$override.description } elseif ($isDetailedFamily) { "Tenant-scoped $family operation. The backend is authoritative for RBAC, resource visibility, transition state and validation." } else { 'Generated router contract. Request and response field detail is pending endpoint-level schema review.' }
        tags = $tag
        'x-contract-status' = if ($override -and $override.status) { [string]$override.status } elseif ($isDetailedFamily) { 'detailed' } else { 'generated' }
        responses = [ordered]@{}
    }
    $successStatus = if ($exactEmptyResponse) { '204' } elseif ($override -and $override.responseStatus) { [string]$override.responseStatus } else { '200' }
    $responseContentType = if ($override -and $override.responseContentType) { [string]$override.responseContentType } else { 'application/json' }
    $successResponse = [ordered]@{ description = 'Successful response' }
    if ($successStatus -ne '204') {
        $successResponse.content = [ordered]@{ $responseContentType = [ordered]@{ schema = [ordered]@{ '$ref' = "#/components/schemas/$responseSchema" } } }
    }
    $operation.responses[$successStatus] = $successResponse
    $errorResponseMap = @{
        '400' = '#/components/responses/BadRequest'; '401' = '#/components/responses/Unauthorized'
        '403' = '#/components/responses/Forbidden'; '404' = '#/components/responses/NotFound'
        '422' = '#/components/responses/Validation'; '500' = '#/components/responses/ServerError'
    }
    $errorStatuses = if ($override -and $override.errors) { @($override.errors | ForEach-Object { [string]$_ }) } else { @('400','401','403','404','422','500') }
    foreach ($errorStatus in $errorStatuses) {
        if ($errorResponseMap.ContainsKey($errorStatus)) { $operation.responses[$errorStatus] = [ordered]@{ '$ref' = $errorResponseMap[$errorStatus] } }
    }

    $requiresSecurity = -not $isPublic
    if ($override -and $override.security -eq 'none') { $requiresSecurity = $false }
    if ($requiresSecurity) {
        if ($override -and $override.security -eq 'refreshCookie') {
            $operation.security = @(@{ refreshCookie = @() })
        } elseif ($override -and $override.security -eq 'productionE2ECanaryActivation') {
            $operation.security = @(@{ productionE2ECanaryActivation = @() })
        } else {
            $operation.security = @(@{ oidcAuthorizationCode = @() })
        }
        $operation.'x-tenant-scope' = if ($override -and $null -ne $override.tenantScope) { [bool]$override.tenantScope } else { $true }
    } else {
        $operation.security = @()
    }
    if ($override -and $override.requiredPermission) { $operation.'x-required-permission' = [string]$override.requiredPermission }
    elseif ($permission) { $operation.'x-required-permission' = $permission }
    elseif ($requiresSecurity) { $operation.'x-required-permission' = 'authenticated' }

    $parameters = @()
    $pathParameterNames = if ($override -and $override.pathParameters) { @($override.pathParameters) } else { @([regex]::Matches($path, '\{([^}]+)\}') | ForEach-Object { $_.Groups[1].Value }) }
    foreach ($parameterName in $pathParameterNames) {
        $parameters += [ordered]@{ name = [string]$parameterName; in = 'path'; required = $true; schema = [ordered]@{ type = 'string' } }
    }
    if ($override -and $override.institutionContext) { $parameters += [ordered]@{ '$ref' = '#/components/parameters/Institution' } }

    $queryParameterNames = if ($override -and $override.queryParameters) { @($override.queryParameters) } else { @() }
    foreach ($parameterName in $queryParameterNames) {
        if ($parameterName -eq 'page') { $parameters += [ordered]@{ '$ref' = '#/components/parameters/Page' }; continue }
        if ($parameterName -eq 'pageSize') { $parameters += [ordered]@{ '$ref' = '#/components/parameters/PageSize' }; continue }
        $parameters += [ordered]@{ name = [string]$parameterName; in = 'query'; required = $false; schema = [ordered]@{ type = 'string' } }
    }
    if ($parameters.Count -gt 0) { $operation.parameters = $parameters }
    $hasRequestBody = -not ($override -and $override.Contains('requestBody') -and -not [bool]$override.requestBody)
    # Activation is an authenticated, bodyless command.  The handler does not
    # decode JSON, so documenting a required empty object would reject valid
    # clients and create a false SDK method signature.
    if ($key -eq 'POST /api/eudi-wallet/activate') { $hasRequestBody = $false }
    if ($method -in @('post', 'put', 'patch') -and $hasRequestBody) {
        $requestSchema = if ($override -and $override.requestSchema) { [string]$override.requestSchema } elseif ($isDetailedFamily) { "$family`Request" } else { 'Mutation' }
        if ($requestSchema -in @('IdentityRequest', 'AdminCommand', 'GdprCommand', 'Mutation', 'RegistraturaRequest')) {
            $requestSchema = "Request_$($operation.operationId)"
            if (-not $common.components.schemas.Contains($requestSchema)) { $common.components.schemas[$requestSchema] = New-ClosedRequestSchema $key }
        }
        $requestContentType = if ($override -and $override.requestContentType) { [string]$override.requestContentType } else { 'application/json' }
        $operation.requestBody = [ordered]@{ required = $true; content = [ordered]@{ $requestContentType = [ordered]@{ schema = [ordered]@{ '$ref' = "#/components/schemas/$requestSchema" } } } }
    }
    if (-not $paths.Contains($path)) { $paths[$path] = [ordered]@{} }
    $paths[$path][$method] = $operation
    $operations += [ordered]@{ method = $method.ToUpperInvariant(); path = $path; operationId = $operation.operationId }
}

# OIDC is mounted through chi.Handle, so its individual standard endpoints are declared here.
function New-JsonResponse([string]$schema, [string]$description = 'Successful response') {
    return [ordered]@{ description = $description; content = [ordered]@{ 'application/json' = [ordered]@{ schema = [ordered]@{ '$ref' = "#/components/schemas/$schema" } } } }
}
$paths['/api/oidc/.well-known/openid-configuration'] = [ordered]@{ get = [ordered]@{
    operationId = 'get_oidc_discovery'; summary = 'OpenID Connect discovery document'; tags = @('OIDC Provider'); security = @(); 'x-contract-status' = 'detailed'; responses = [ordered]@{ '200' = (New-JsonResponse 'OidcDiscoveryDocument') }
} }
$authorizationParameters = @(
    [ordered]@{ name='client_id'; in='query'; required=$true; schema=[ordered]@{type='string'} }, [ordered]@{ name='redirect_uri'; in='query'; required=$true; schema=[ordered]@{type='string';format='uri'} },
    [ordered]@{ name='response_type'; in='query'; required=$true; schema=[ordered]@{type='string';enum=@('code')} }, [ordered]@{ name='code_challenge'; in='query'; required=$true; schema=[ordered]@{type='string'} },
    [ordered]@{ name='code_challenge_method'; in='query'; required=$true; schema=[ordered]@{type='string';enum=@('S256')} }, [ordered]@{ name='state'; in='query'; required=$true; schema=[ordered]@{type='string'} }, [ordered]@{ name='nonce'; in='query'; required=$true; schema=[ordered]@{type='string'} }
)
$paths['/api/oidc/authorize'] = [ordered]@{ get = [ordered]@{
    operationId='get_oidc_authorize'; summary='OIDC authorization endpoint'; description='Standards-compliant authorization endpoint. Public clients must send PKCE S256, state and nonce.'; tags=@('OIDC Provider'); security=@(); 'x-contract-status'='detailed'; parameters=$authorizationParameters; responses=[ordered]@{'302'=[ordered]@{description='Redirect to client callback after interaction'}}
} }
$tokenSchema = [ordered]@{ type='object'; required=@('grant_type'); properties=[ordered]@{ grant_type=[ordered]@{type='string';enum=@('authorization_code','refresh_token')}; code=[ordered]@{type='string'}; code_verifier=[ordered]@{type='string'}; redirect_uri=[ordered]@{type='string';format='uri'}; refresh_token=[ordered]@{type='string'} } }
$paths['/api/oidc/token'] = [ordered]@{ post = [ordered]@{
    operationId='post_oidc_token'; summary='OIDC token endpoint'; description='Exchanges an authorization code with code_verifier, or refreshes a session. Never log codes, refresh tokens or DPoP proofs.'; tags=@('OIDC Provider'); security=@(); 'x-contract-status'='detailed'; requestBody=[ordered]@{required=$true;content=[ordered]@{'application/x-www-form-urlencoded'=[ordered]@{schema=$tokenSchema}}}; responses=[ordered]@{'200'=(New-JsonResponse 'OidcTokenResponse');'400'=[ordered]@{'$ref'='#/components/responses/BadRequest'}}
} }
$revocationSchema = [ordered]@{ type='object'; required=@('token','client_id'); properties=[ordered]@{ token=[ordered]@{type='string';description='The actual refresh or access token to revoke. The literal value `cookie` is not accepted.'}; token_type_hint=[ordered]@{type='string';enum=@('refresh_token','access_token')}; client_id=[ordered]@{type='string'} } }
$paths['/api/oidc/revoke'] = [ordered]@{ post = [ordered]@{
    operationId='post_oidc_revoke'; summary='RFC 7009 token revocation endpoint'; description='Revokes a token issued to an allow-listed first-party client. This endpoint accepts the actual token only; it never substitutes an HttpOnly cookie value.'; tags=@('OIDC Provider'); security=@(); 'x-contract-status'='detailed'; requestBody=[ordered]@{required=$true;content=[ordered]@{'application/x-www-form-urlencoded'=[ordered]@{schema=$revocationSchema}}}; responses=[ordered]@{'200'=[ordered]@{description='Revocation processed, including unknown or already-revoked tokens per RFC 7009.'};'400'=[ordered]@{'$ref'='#/components/responses/BadRequest'};'401'=[ordered]@{'$ref'='#/components/responses/Unauthorized'};'403'=[ordered]@{'$ref'='#/components/responses/Forbidden'}}
} }
$rpLogoutParameters = @(
    [ordered]@{name='id_token_hint';in='query';required=$false;schema=[ordered]@{type='string'}},
    [ordered]@{name='post_logout_redirect_uri';in='query';required=$false;schema=[ordered]@{type='string';format='uri'}},
    [ordered]@{name='state';in='query';required=$false;schema=[ordered]@{type='string'}},
    [ordered]@{name='client_id';in='query';required=$false;schema=[ordered]@{type='string'}}
)
$paths['/api/oidc/session/end'] = [ordered]@{ get = [ordered]@{
    operationId='get_oidc_session_end'; summary='OpenID Connect RP-initiated logout'; description='Ends the provider refresh session synchronously and redirects only to an exact `post_logout_redirect_uri` registered for the resolved client. If supplied, state is preserved on that redirect.'; tags=@('OIDC Provider'); security=@(); 'x-contract-status'='detailed'; parameters=$rpLogoutParameters; responses=[ordered]@{'302'=[ordered]@{description='Registered RP post-logout redirect or the configured frontend fallback.'};'400'=[ordered]@{'$ref'='#/components/responses/BadRequest'}}
}; post = [ordered]@{
    operationId='post_oidc_session_end'; summary='OpenID Connect RP-initiated logout'; description='Form-post variant of RP-initiated logout with the same exact redirect-uri validation.'; tags=@('OIDC Provider'); security=@(); 'x-contract-status'='detailed'; requestBody=[ordered]@{required=$false;content=[ordered]@{'application/x-www-form-urlencoded'=[ordered]@{schema=[ordered]@{type='object';properties=[ordered]@{id_token_hint=[ordered]@{type='string'};post_logout_redirect_uri=[ordered]@{type='string';format='uri'};state=[ordered]@{type='string'};client_id=[ordered]@{type='string'}}}}}}; responses=[ordered]@{'302'=[ordered]@{description='Registered RP post-logout redirect or the configured frontend fallback.'};'400'=[ordered]@{'$ref'='#/components/responses/BadRequest'}}
} }
$paths['/api/oidc/jwks'] = [ordered]@{ get = [ordered]@{
    operationId='get_oidc_jwks'; summary='OIDC JSON Web Key Set'; tags=@('OIDC Provider'); security=@(); 'x-contract-status'='detailed'; responses=[ordered]@{'200'=(New-JsonResponse 'JsonWebKeySet')}
} }

$document = [ordered]@{
    openapi = '3.1.1'
    info = [ordered]@{ title = 'EguEducation API'; version = '1.0.0'; description = 'Tenant-aware EguEducation backend contract. Generated from the server router and handler-backed domain catalogs. The backend, not a browser-controlled header, derives the active tenant from authenticated membership, token/session and host context.'; license = [ordered]@{ name = 'Proprietary — EguEducation'; identifier = 'LicenseRef-EguEducation-Proprietary' } }
    servers = @([ordered]@{ url = '/'; description = 'Current deployment origin' })
    tags = @(
        [ordered]@{name='Authentication'; description='Current-session profile, WebAuthn credentials and authenticated application identity.'},
        [ordered]@{name='OIDC Provider'; description='Standards-based OpenID Connect provider endpoints, including PKCE authorization-code exchange.'},
        [ordered]@{name='Registratura'; description='Incoming, outgoing and internal document registration and registers.'},
        [ordered]@{name='Workflow'; description='Document workflow tasks, transitions and audit-visible state.'},
        [ordered]@{name='eArhiva'; description='Tenant-scoped electronic archive records and documents.'},
        [ordered]@{name='Scoala'; description='Institution-scoped school operational records; tenant context is server-derived.'},
        [ordered]@{name='Administrare'; description='Tenant administration, RBAC, modules, OIDC client metadata and configuration.'},
        [ordered]@{name='GDPR'; description='Retention, subject access, export and publication-review administration.'},
        [ordered]@{name='Platform'; description='Deployment health and public bootstrap configuration.'}
    )
    paths = $paths
    components = $common.components
}

# Domain catalogues intentionally carry reusable Go-model projections. Keep only
# components reachable from an operation (and their transitive schema references)
# in the published contract so the document cannot advertise dead DTOs.
$reachableSchemas = @{}
$reachableSchemas['Problem'] = $true
$pathJson = $document.paths | ConvertTo-Json -Depth 100
foreach ($reference in [regex]::Matches($pathJson, '#/components/schemas/([^"/]+)')) { $reachableSchemas[$reference.Groups[1].Value] = $true }
$changed = $true
while ($changed) {
    $changed = $false
    foreach ($schemaName in @($reachableSchemas.Keys)) {
        if (-not $document.components.schemas.Contains($schemaName)) { continue }
        $schemaJson = $document.components.schemas[$schemaName] | ConvertTo-Json -Depth 100
        foreach ($reference in [regex]::Matches($schemaJson, '#/components/schemas/([^"/]+)')) {
            $referencedSchema = $reference.Groups[1].Value
            if (-not $reachableSchemas.Contains($referencedSchema)) { $reachableSchemas[$referencedSchema] = $true; $changed = $true }
        }
    }
}
foreach ($schemaName in @($document.components.schemas.Keys)) {
    if (-not $reachableSchemas.Contains($schemaName)) { $document.components.schemas.Remove($schemaName) }
}

$outputDirectory = Split-Path -Parent $Output
New-Item -ItemType Directory -Force -Path $outputDirectory | Out-Null
$document | ConvertTo-Json -Depth 100 | Set-Content -NoNewline -Encoding utf8 $Output
$operations | ConvertTo-Json -Depth 10 | Set-Content -NoNewline -Encoding utf8 'openapi/router-operations.json'
& node scripts/openapi/canonicalize-json.js $Output 'openapi/router-operations.json'
if ($LASTEXITCODE -ne 0) { throw 'OpenAPI JSON canonicalization failed.' }
New-Item -ItemType Directory -Force -Path 'backend/internal/apidocs' | Out-Null
Copy-Item -Force $Output 'backend/internal/apidocs/openapi.json'
Write-Host "Generated $($operations.Count) concrete router operations in $Output"
