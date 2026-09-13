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
    if ($path -like '/api/admissions/*') { return 'Scoala' }
    if ($path -like '/api/institution/*') { return 'Institutie' }
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
    if ($path -like '/api/institution/*') { return 'Institution' }
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
		'POST /api/regulatory-sources'='backend/internal/regulatorysource/models.go|RegisterRegulatorySourceRequest'
		'POST /api/regulatory-sources/{sourceID}/verify'='backend/internal/regulatorysource/models.go|VerifyRegulatorySourceRequest'
		'POST /api/regulatory-sources/{sourceID}/activate'='backend/internal/regulatorysource/models.go|ActivateRegulatorySourceRequest'
		'POST /api/earchiva/retention-rules'='backend/internal/earchiva/archive_series_retention.go|ProposeArchiveSeriesRetentionRuleRequest'
		'POST /api/earchiva/retention-rules/{ruleID}/approve'='backend/internal/earchiva/archive_series_retention.go|ApproveArchiveSeriesRetentionRuleRequest'
		'POST /api/earchiva/retention-rules/{ruleID}/retire'='backend/internal/earchiva/archive_series_retention.go|RetireArchiveSeriesRetentionRuleRequest'
		'POST /api/admissions/applications/{applicationID}/decision-preparations'='backend/internal/admission/models.go|PrepareDecisionRequest'
		'POST /api/admissions/appeals/{appealID}/resolution-preparations'='backend/internal/admission/models.go|PrepareAppealResolutionRequest'
		'POST /api/admissions/legal-preparations/finalize'='backend/internal/admission/models.go|FinalizeAdmissionLegalPreparationRequest'
		'POST /api/admissions/signer-authorizations'='backend/internal/admission/models.go|ProposeAdmissionSignerAuthorizationRequest'
		'POST /api/admissions/signer-authorizations/approve'='backend/internal/admission/models.go|ApproveAdmissionSignerAuthorizationRequest'
		'POST /api/admissions/signer-authorizations/{authorizationID}/revoke'='backend/internal/admission/models.go|RevokeAdmissionSignerAuthorizationRequest'
		'POST /api/passkeys/login-finish'='backend/internal/auth/models.go|FinishPasskeyAuthenticationRequest'; 'POST /api/passkeys/register-finish'='backend/internal/auth/models.go|FinishPasskeyRegistrationRequest'; 'PUT /api/profile'='backend/internal/auth/models.go|UpdateProfileRequest'
		'POST /api/admin/users'='backend/internal/admin/models.go|UpsertUserRequest'; 'POST /api/admin/roles'='backend/internal/admin/models.go|UpsertRoleRequest'; 'POST /api/admin/role-assignments'='backend/internal/admin/models.go|UpsertUserRoleAssignmentRequest'; 'POST /api/admin/role-permissions'='backend/internal/admin/models.go|UpsertRolePermissionAssignmentRequest'; 'POST /api/admin/position-roles'='backend/internal/admin/models.go|UpsertPositionRoleAssignmentRequest'; 'POST /api/admin/org-units'='backend/internal/admin/models.go|UpsertOrgUnitRequest'; 'POST /api/admin/memberships'='backend/internal/admin/models.go|UpsertMembershipRequest'; 'POST /api/admin/positions'='backend/internal/admin/models.go|UpsertPositionRequest'; 'POST /api/admin/permissions/assignments'='backend/internal/admin/models.go|UpsertPermissionAssignmentRequest'; 'POST /api/admin/auth-methods'='backend/internal/admin/models.go|UpdateAuthMethodSettingRequest'; 'POST /api/admin/modules'='backend/internal/admin/models.go|UpdateModuleSettingRequest'; 'POST /api/admin/oidc/clients'='backend/internal/admin/models.go|UpsertOIDCClientRequest'; 'POST /api/admin/gdpr-settings'='backend/internal/admin/models.go|UpdateGdprSettingRequest'; 'POST /api/admin/dossier-requirements'='backend/internal/admin/models.go|CreateDossierRequirementRequest'; 'POST /api/admin/workflow-definitions'='backend/internal/admin/models.go|CreateWorkflowDefinitionRequest'; 'POST /api/admin/nomenclatures'='backend/internal/admin/models.go|CreateNomenclatureRequest'; 'POST /api/admin/education-taxonomies'='backend/internal/admin/models.go|CreateEducationTaxonomyRequest'
		'POST /api/registratura/documents'='backend/internal/registratura/models.go|CreateDocumentRequest'; 'PATCH /api/registratura/documents/{documentID}'='backend/internal/registratura/models.go|UpdateDocumentRequest'; 'POST /api/registratura/documents/batch'='backend/internal/registratura/models.go|BatchCreateDocumentsRequest'; 'POST /api/registratura/documents/export-pdf'='backend/internal/registratura/models.go|ExportDocumentsRequest'; 'POST /api/registratura/documents/{documentID}/versions'='backend/internal/registratura/models.go|CreateDocumentVersionRequest'; 'POST /api/registratura/documents/{documentID}/attachments'='backend/internal/registratura/models.go|CreateDocumentAttachmentRequest'; 'POST /api/registratura/registre'='backend/internal/registratura/models.go|CreateRegistruRequest'; 'PATCH /api/registratura/registre/{id}'='backend/internal/registratura/models.go|UpdateRegistruRequest'; 'POST /api/registratura/parties'='backend/internal/registratura/models.go|CreatePartyRequest'; 'PATCH /api/registratura/parties/{id}'='backend/internal/registratura/models.go|UpdatePartyRequest'; 'POST /api/registratura/admin/departments'='backend/internal/registratura/structure.go|departmentRequest'; 'PATCH /api/registratura/admin/departments/{id}'='backend/internal/registratura/structure.go|departmentRequest'; 'POST /api/registratura/admin/organizations'='backend/internal/registratura/structure.go|organizationRequest'; 'PATCH /api/registratura/admin/organizations/{id}'='backend/internal/registratura/structure.go|organizationRequest'; 'PUT /api/registratura/admin/users/{id}/assignments'='backend/internal/registratura/structure.go|assignmentRequest'; 'POST /api/registratura/admin/registries'='backend/internal/registratura/structure.go|adminRegistryRequest'; 'PATCH /api/registratura/admin/registries/{id}'='backend/internal/registratura/structure.go|adminRegistryRequest'; 'POST /api/registratura/document-links'='backend/internal/registratura/models.go|CreateDocumentLinkRequest'
		'POST /api/gdpr/retention-policies'='backend/internal/gdpr/models.go|CreateRetentionPolicyRequest'; 'POST /api/gdpr/subject-requests'='backend/internal/gdpr/models.go|CreateSubjectRequestRequest'; 'POST /api/gdpr/exports'='backend/internal/gdpr/models.go|CreateSubjectExportRequest'; 'POST /api/gdpr/publication-reviews'='backend/internal/gdpr/models.go|CreatePublicationReviewRequest'
		'POST /api/earchiva/classification-reviews/{reviewID}/approve'='backend/internal/earchiva/archive_classification.go|ArchiveClassificationApprovalRequest'; 'POST /api/earchiva/classification-reviews/{reviewID}/correct'='backend/internal/earchiva/archive_classification.go|ArchiveClassificationCorrectionRequest'
		'POST /api/earchiva/admin/portfolio-custody-intents/{intentID}/reconcile'='backend/internal/earchiva/portfolio_custody_recovery.go|ReconcilePortfolioCustodyRequest'
		'POST /api/education/portfolios/me'='backend/internal/education/portfolio_models.go|OwnPortfolioRequest'; 'PATCH /api/education/portfolios/me/{recordID}'='backend/internal/education/portfolio_models.go|OwnPortfolioRequest'; 'POST /api/education/portfolios/me/{recordID}/documents'='backend/internal/education/portfolio_models.go|OwnPortfolioDocumentRequest'; 'PATCH /api/education/portfolios/me/{recordID}/documents/{documentID}'='backend/internal/education/portfolio_models.go|OwnPortfolioDocumentRequest'; 'POST /api/education/portfolios/me/{recordID}/declarations/{declarationType}/acknowledgements'='backend/internal/education/portfolio_declarations.go|PortfolioDeclarationAcknowledgementRequest'; 'POST /api/education/portfolios/records/{recordID}/activity-cessation'='backend/internal/education/portfolio_models.go|PortfolioCessationRequest'; 'POST /api/education/portfolios/records/{recordID}/legal-hold'='backend/internal/education/portfolio_models.go|PortfolioLegalHoldRequest'
		'POST /api/education/portfolios/records/{recordID}/return'='backend/internal/education/governance_portfolio_flows.go|PortfolioReturnForCorrectionsRequest'; 'POST /api/education/portfolios/records/{recordID}/managerial-decision'='backend/internal/education/governance_portfolio_flows.go|PortfolioManagerialDecisionRequest'
		'PUT /api/institution/regulatory-profile'='backend/internal/institution/models.go|PutRegulatoryProfileRequest'; 'POST /api/institution/locations'='backend/internal/institution/models.go|CreateSchoolLocationRequest'; 'PATCH /api/institution/locations/{locationID}'='backend/internal/institution/models.go|UpdateSchoolLocationRequest'; 'POST /api/institution/education-offerings'='backend/internal/institution/models.go|CreateEducationOfferingRequest'; 'PATCH /api/institution/education-offerings/{offeringID}'='backend/internal/institution/models.go|UpdateEducationOfferingRequest'; 'POST /api/institution/offering-authorizations'='backend/internal/institution/models.go|CreateOfferingAuthorizationRequest'
		'POST /api/school-operations/contracts'='backend/internal/schooloperations/models.go|CreateContractRequest'; 'PATCH /api/school-operations/contracts/{contractID}'='backend/internal/schooloperations/models.go|AmendContractRequest'; 'POST /api/school-operations/contracts/{contractID}/obligations'='backend/internal/schooloperations/models.go|CreateContractObligationRequest'; 'POST /api/school-operations/contracts/{contractID}/transition'='backend/internal/schooloperations/models.go|TransitionContractRequest'
		'POST /api/admissions/campaigns'='backend/internal/admission/models.go|CreateCampaignRequest'; 'POST /api/admissions/campaigns/{campaignID}/transitions'='backend/internal/admission/models.go|TransitionRequest'; 'POST /api/admissions/campaigns/{campaignID}/criteria'='backend/internal/admission/models.go|CriterionInput'; 'POST /api/admissions/campaigns/{campaignID}/document-requirements'='backend/internal/admission/models.go|DocumentRequirementInput'
		'POST /api/admissions/class-offering-contexts'='backend/internal/admission/models.go|CreateClassOfferingContextRequest'
		'POST /api/admissions/applications'='backend/internal/admission/models.go|CreateApplicationRequest'; 'POST /api/admissions/applications/{applicationID}/transitions'='backend/internal/admission/models.go|TransitionRequest'; 'POST /api/admissions/applications/{applicationID}/assessments'='backend/internal/admission/models.go|AssessCriterionRequest'; 'POST /api/admissions/applications/{applicationID}/documents/{documentID}'='backend/internal/admission/models.go|ReviewApplicationDocumentRequest'; 'POST /api/admissions/applications/{applicationID}/decisions'='backend/internal/admission/models.go|IssueDecisionRequest'; 'POST /api/admissions/applications/{applicationID}/appeals'='backend/internal/admission/models.go|CreateAppealRequest'; 'POST /api/admissions/applications/{applicationID}/enrolment'='backend/internal/admission/models.go|EnrolApplicationRequest'; 'POST /api/admissions/appeals/{appealID}/resolution'='backend/internal/admission/models.go|ResolveAppealRequest'; 'POST /api/admissions/retention-rule-versions'='backend/internal/admission/dss_retention_policy.go|ProposeAdmissionRetentionRuleRequest'; 'POST /api/admissions/retention-rule-versions/approve'='backend/internal/admission/dss_retention_policy.go|ApproveAdmissionRetentionRuleRequest'; 'POST /api/admissions/dss-retention-policies'='backend/internal/admission/dss_retention_policy.go|ConfigureDSSRetentionPolicyRequest'
	}
	if ($dtoMap.ContainsKey($operationKey)) {
		return Get-GoDTOObjectSchema ([string]$dtoMap[$operationKey])
	}
	if ($operationKey -eq 'POST /api/admissions/legal-preparations/{preparationID}/artifacts/{artifactSlot}') {
		return [ordered]@{
			type = 'object'; additionalProperties = $false; required = @('file')
			properties = [ordered]@{ file = [ordered]@{ type = 'string'; format = 'binary' } }
		}
	}
	if ($operationKey -eq 'POST /api/education/portfolios/me/{recordID}/archive-documents') {
		return [ordered]@{
			type='object'; additionalProperties=$false; required=@('file','title')
			properties=[ordered]@{
				file=[ordered]@{type='string';format='binary'}
				title=[ordered]@{type='string';minLength=1;maxLength=300}
				document_date=[ordered]@{type='string';format='date'}
			}
		}
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
		'GET /api/earchiva/retention-rules'='page|backend/internal/earchiva/archive_series_retention.go|ArchiveSeriesRetentionRule'
		'GET /api/regulatory-sources'='page|backend/internal/regulatorysource/models.go|RegulatorySource'
		'POST /api/education/portfolios/me/{recordID}/archive-documents'='dto|backend/internal/earchiva/archive_documents.go|ArchiveDocumentDetail'
		'POST /api/regulatory-sources'='dto|backend/internal/regulatorysource/models.go|RegulatorySource'
		'POST /api/regulatory-sources/{sourceID}/verify'='dto|backend/internal/regulatorysource/models.go|RegulatorySource'
		'POST /api/regulatory-sources/{sourceID}/activate'='dto|backend/internal/regulatorysource/models.go|RegulatorySource'
		'POST /api/earchiva/retention-rules'='dto|backend/internal/earchiva/archive_series_retention.go|ArchiveSeriesRetentionRule'
		'POST /api/earchiva/retention-rules/{ruleID}/approve'='dto|backend/internal/earchiva/archive_series_retention.go|ArchiveSeriesRetentionRule'
		'POST /api/earchiva/retention-rules/{ruleID}/retire'='dto|backend/internal/earchiva/archive_series_retention.go|ArchiveSeriesRetentionRule'
		'POST /api/admissions/legal-preparations/{preparationID}/artifacts/{artifactSlot}'='dto|backend/internal/earchiva/admission_legal_upload.go|AdmissionLegalPreparationArtifactResponse'
		'GET /api/admissions/legal-preparations/{preparationID}/artifacts/{artifactSlot}'='dto|backend/internal/earchiva/admission_legal_upload.go|AdmissionLegalPreparationArtifactResponse'
		'POST /api/admissions/applications/{applicationID}/decision-preparations'='dto|backend/internal/admission/models.go|AdmissionLegalPreparation'
		'POST /api/admissions/appeals/{appealID}/resolution-preparations'='dto|backend/internal/admission/models.go|AdmissionLegalPreparation'
		'POST /api/admissions/legal-preparations/finalize'='dto|backend/internal/admission/models.go|CommandResult'
		'POST /api/admissions/signer-authorizations'='dto|backend/internal/admission/models.go|AdmissionSignerAuthorization'
		'POST /api/admissions/signer-authorizations/approve'='dto|backend/internal/admission/models.go|AdmissionSignerAuthorization'
		'POST /api/admissions/signer-authorizations/{authorizationID}/revoke'='dto|backend/internal/admission/models.go|CommandResult'
		'POST /api/admissions/legal-preparations/{preparationID}/cancel'='dto|backend/internal/admission/models.go|CommandResult'
		'GET /api/admissions/signer-authorizations'='page|backend/internal/admission/models.go|AdmissionSignerAuthorization'
        'GET /api/auth/methods'='object|auth_methods'; 'GET /api/auth/ui-config'='object|auth_ui_config'; 'GET /api/auth/role-catalog'='dto|backend/internal/auth/models.go|RoleCatalogResponse'; 'GET /api/auth/role-positions'='dto|backend/internal/auth/models.go|RolePositionResponse'
        'POST /api/passkeys/login-options'='object|passkey_login_options'; 'POST /api/passkeys/login-finish'='object|passkey_login_finish'; 'POST /api/passkeys/register-options'='object|passkey_register_options'; 'POST /api/passkeys/register-finish'='dto|backend/internal/auth/models.go|PasskeyCredentialSummary'; 'POST /api/eudi-wallet/activate'='object|eudi_activation'
        'GET /api/passkeys'='array|backend/internal/auth/models.go|PasskeyCredentialSummary'
		'GET /api/institution/regulatory-profile'='dto|backend/internal/institution/models.go|RegulatoryProfile'; 'PUT /api/institution/regulatory-profile'='dto|backend/internal/institution/models.go|RegulatoryProfile'; 'GET /api/institution/capabilities'='dto|backend/internal/institution/models.go|InstitutionCapabilitiesResponse'; 'GET /api/institution/policy-cutover-preflight'='dto|backend/internal/institution/models.go|PolicyCutoverPreflightResponse'; 'GET /api/institution/locations'='page|backend/internal/institution/models.go|SchoolLocation'; 'POST /api/institution/locations'='dto|backend/internal/institution/models.go|SchoolLocation'; 'PATCH /api/institution/locations/{locationID}'='dto|backend/internal/institution/models.go|SchoolLocation'; 'GET /api/institution/education-offerings'='page|backend/internal/institution/models.go|EducationOffering'; 'POST /api/institution/education-offerings'='dto|backend/internal/institution/models.go|EducationOffering'; 'PATCH /api/institution/education-offerings/{offeringID}'='dto|backend/internal/institution/models.go|EducationOffering'; 'GET /api/institution/offering-authorizations'='page|backend/internal/institution/models.go|OfferingAuthorization'; 'POST /api/institution/offering-authorizations'='dto|backend/internal/institution/models.go|OfferingAuthorization'
		'GET /api/school-operations/suppliers'='page|backend/internal/schooloperations/models.go|SupplierOption'
		'GET /api/school-operations/contracts'='page|backend/internal/schooloperations/models.go|Contract'; 'POST /api/school-operations/contracts'='object|school_operations_contract_create'; 'GET /api/school-operations/contracts/{contractID}'='dto|backend/internal/schooloperations/models.go|Contract'; 'PATCH /api/school-operations/contracts/{contractID}'='object|school_operations_contract_amend'; 'GET /api/school-operations/contracts/{contractID}/obligations'='page|backend/internal/schooloperations/models.go|ContractObligation'; 'POST /api/school-operations/contracts/{contractID}/obligations'='object|school_operations_contract_obligation_create'; 'POST /api/school-operations/contracts/{contractID}/transition'='object|school_operations_contract_transition'
		'GET /api/admissions/campaigns'='page|backend/internal/admission/models.go|Campaign'; 'POST /api/admissions/campaigns'='dto|backend/internal/admission/models.go|CommandResult'; 'POST /api/admissions/campaigns/{campaignID}/transitions'='dto|backend/internal/admission/models.go|CommandResult'; 'GET /api/admissions/campaigns/{campaignID}/criteria'='page|backend/internal/admission/models.go|Criterion'; 'POST /api/admissions/campaigns/{campaignID}/criteria'='dto|backend/internal/admission/models.go|CommandResult'; 'GET /api/admissions/campaigns/{campaignID}/document-requirements'='page|backend/internal/admission/models.go|DocumentRequirement'; 'POST /api/admissions/campaigns/{campaignID}/document-requirements'='dto|backend/internal/admission/models.go|CommandResult'
		'GET /api/admissions/class-offering-contexts'='page|backend/internal/admission/models.go|ClassOfferingContext'; 'POST /api/admissions/class-offering-contexts'='dto|backend/internal/admission/models.go|CommandResult'; 'GET /api/admissions/regulatory-sources'='page|backend/internal/admission/models.go|RegulatorySourceOption'; 'GET /api/admissions/candidate-parties'='page|backend/internal/admission/models.go|CandidatePartyOption'; 'GET /api/admissions/students'='page|backend/internal/admission/models.go|StudentOption'; 'GET /api/admissions/eligible-archive-versions'='page|backend/internal/admission/models.go|ArchiveVersionOption'
		'GET /api/admissions/applications'='page|backend/internal/admission/models.go|Application'; 'POST /api/admissions/applications'='dto|backend/internal/admission/models.go|CommandResult'; 'GET /api/admissions/applications/{applicationID}'='dto|backend/internal/admission/models.go|ApplicationDetail'; 'POST /api/admissions/applications/{applicationID}/transitions'='dto|backend/internal/admission/models.go|CommandResult'; 'POST /api/admissions/applications/{applicationID}/assessments'='dto|backend/internal/admission/models.go|CommandResult'; 'POST /api/admissions/applications/{applicationID}/documents/{documentID}'='dto|backend/internal/admission/models.go|CommandResult'; 'POST /api/admissions/applications/{applicationID}/decisions'='dto|backend/internal/admission/models.go|Decision'; 'POST /api/admissions/applications/{applicationID}/appeals'='dto|backend/internal/admission/models.go|CommandResult'; 'POST /api/admissions/applications/{applicationID}/enrolment'='dto|backend/internal/admission/models.go|CommandResult'; 'GET /api/admissions/decisions'='page|backend/internal/admission/models.go|Decision'; 'GET /api/admissions/appeals'='page|backend/internal/admission/models.go|Appeal'; 'POST /api/admissions/appeals/{appealID}/resolution'='dto|backend/internal/admission/models.go|CommandResult'
		'GET /api/registratura/documents/filters'='dto|backend/internal/registratura/models.go|DocumentFiltersResponse'; 'GET /api/registratura/nomenclatures'='dto|backend/internal/registratura/models.go|DocumentFiltersResponse'; 'POST /api/registratura/documents'='dto|backend/internal/registratura/models.go|Document'; 'PATCH /api/registratura/documents/{documentID}'='dto|backend/internal/registratura/models.go|Document'; 'GET /api/registratura/documents/{documentID}'='dto|backend/internal/registratura/models.go|Document'; 'POST /api/registratura/documents/{documentID}/cancel'='dto|backend/internal/registratura/models.go|Document'; 'GET /api/registratura/documents/lookup'='array|backend/internal/registratura/models.go|DocumentLookupItem'; 'GET /api/registratura/documents/{documentID}/versions'='array|backend/internal/registratura/models.go|DocumentVersion'; 'POST /api/registratura/documents/{documentID}/versions'='dto|backend/internal/registratura/models.go|DocumentVersion'; 'GET /api/registratura/documents/{documentID}/workflow-history'='array|backend/internal/registratura/models.go|DocumentWorkflowEvent'; 'GET /api/registratura/workflow-assignees'='object|workflow_assignees'; 'GET /api/registratura/documents/{documentID}/attachments'='array|backend/internal/registratura/models.go|DocumentAttachment'; 'POST /api/registratura/documents/{documentID}/attachments'='dto|backend/internal/registratura/models.go|DocumentAttachment'; 'GET /api/registratura/flux/queue'='page|backend/internal/registratura/models.go|FluxDocument'; 'GET /api/registratura/flux/mapa'='page|backend/internal/registratura/models.go|FluxDocument'; 'GET /api/registratura/flux/pipeline'='page|backend/internal/registratura/models.go|FluxDocument'; 'GET /api/registratura/flux/pipeline/stats'='array|backend/internal/registratura/models.go|FluxPipelineStat'
        'POST /api/registratura/registre'='dto|backend/internal/registratura/models.go|Registru'; 'GET /api/registratura/registre/{id}'='dto|backend/internal/registratura/models.go|Registru'; 'PATCH /api/registratura/registre/{id}'='dto|backend/internal/registratura/models.go|Registru'; 'DELETE /api/registratura/registre/{id}'='empty|'; 'PATCH /api/registratura/registre/{id}/set-default'='dto|backend/internal/registratura/models.go|Registru'
        'GET /api/registratura/parties'='page|backend/internal/registratura/models.go|Party'; 'POST /api/registratura/parties'='dto|backend/internal/registratura/models.go|Party'; 'GET /api/registratura/parties/lookup'='array|backend/internal/registratura/models.go|Party'; 'GET /api/registratura/parties/default-organization'='dto|backend/internal/registratura/models.go|Party'; 'GET /api/registratura/parties/{id}'='dto|backend/internal/registratura/models.go|Party'; 'PATCH /api/registratura/parties/{id}'='dto|backend/internal/registratura/models.go|Party'; 'DELETE /api/registratura/parties/{id}'='empty|'
        'GET /api/registratura/admin/departments'='page|backend/internal/registratura/models.go|Department'; 'POST /api/registratura/admin/departments'='dto|backend/internal/registratura/models.go|Department'; 'PATCH /api/registratura/admin/departments/{id}'='dto|backend/internal/registratura/models.go|Department'; 'DELETE /api/registratura/admin/departments/{id}'='empty|'; 'GET /api/registratura/admin/organizations'='array|backend/internal/registratura/models.go|Organization'; 'POST /api/registratura/admin/organizations'='dto|backend/internal/registratura/models.go|Organization'; 'PATCH /api/registratura/admin/organizations/{id}'='dto|backend/internal/registratura/models.go|Organization'; 'DELETE /api/registratura/admin/organizations/{id}'='empty|'; 'GET /api/registratura/admin/organization-chart'='arrayinline|organization_chart'; 'GET /api/registratura/admin/users/{id}/assignments'='object|user_assignments'; 'PUT /api/registratura/admin/users/{id}/assignments'='object|user_assignments'; 'GET /api/registratura/admin/registries'='page|backend/internal/registratura/structure.go|adminRegistry'; 'POST /api/registratura/admin/registries'='dto|backend/internal/registratura/structure.go|adminRegistry'; 'PATCH /api/registratura/admin/registries/{id}'='dto|backend/internal/registratura/structure.go|adminRegistry'; 'DELETE /api/registratura/admin/registries/{id}'='empty|'; 'GET /api/registratura/document-links'='array|backend/internal/registratura/models.go|LinkedDocument'; 'POST /api/registratura/document-links'='dto|backend/internal/registratura/models.go|LinkedDocument'; 'DELETE /api/registratura/document-links/{linkID}'='empty|'
        'GET /api/workflow/dashboard'='object|workflow_dashboard'; 'GET /api/workflow/definitions'='array|backend/internal/workflow/models.go|Definition'; 'POST /api/workflow/tasks'='dto|backend/internal/workflow/models.go|Task'; 'GET /api/workflow/tasks/filters'='dto|backend/internal/workflow/models.go|FiltersResponse'
		'GET /api/earchiva/dashboard'='object|archive_dashboard'; 'GET /api/earchiva/records/filters'='dto|backend/internal/earchiva/models.go|FiltersResponse'; 'GET /api/earchiva/nomenclatures'='dto|backend/internal/earchiva/models.go|FiltersResponse'; 'GET /api/earchiva/documents/{documentID}'='dto|backend/internal/earchiva/archive_documents.go|ArchiveDocumentDetail'; 'GET /api/earchiva/documents/{documentID}/versions'='array|backend/internal/earchiva/archive_documents.go|ArchiveDocumentVersionSummary'; 'GET /api/earchiva/taxonomy'='array|backend/internal/earchiva/archive_documents.go|ArchiveTaxonomyNode'; 'GET /api/earchiva/admin/health'='dto|backend/internal/earchiva/archive_admin.go|ArchiveAdminHealth'; 'GET /api/earchiva/admin/stats'='dto|backend/internal/earchiva/archive_admin.go|ArchiveAdminStats'; 'GET /api/earchiva/admin/jobs'='dto|backend/internal/earchiva/archive_admin.go|ArchiveAdminJobPage'; 'POST /api/earchiva/admin/jobs/{jobID}/retry'='dto|backend/internal/earchiva/archive_admin.go|ArchiveAdminJob'
		'GET /api/earchiva/classification-reviews'='dto|backend/internal/earchiva/archive_classification.go|ArchiveClassificationReviewPage'; 'POST /api/earchiva/classification-reviews/{reviewID}/approve'='dto|backend/internal/earchiva/archive_classification.go|ArchiveClassificationReview'; 'POST /api/earchiva/classification-reviews/{reviewID}/correct'='dto|backend/internal/earchiva/archive_classification.go|ArchiveClassificationReview'
		'GET /api/earchiva/admin/portfolio-custody-intents'='page|backend/internal/earchiva/portfolio_custody_recovery.go|PortfolioCustodyRecoveryListItem'; 'POST /api/earchiva/admin/portfolio-custody-intents/{intentID}/reconcile'='dto|backend/internal/earchiva/portfolio_custody_recovery.go|PortfolioCustodyRecoveryOperation'; 'GET /api/earchiva/admin/portfolio-custody-intents/{intentID}/recovery-operations/{operationID}'='dto|backend/internal/earchiva/portfolio_custody_recovery.go|PortfolioCustodyRecoveryOperation'
        'GET /api/registratura/documents'='page|backend/internal/registratura/models.go|Document'; 'GET /api/registratura/registre'='page|backend/internal/registratura/models.go|Registru'; 'GET /api/workflow/tasks'='page|backend/internal/workflow/models.go|Task'; 'GET /api/earchiva/records'='page|backend/internal/earchiva/models.go|Record'; 'GET /api/earchiva/documents'='page|backend/internal/earchiva/archive_documents.go|ArchiveDocumentSearchResult'
        'GET /api/education/governance/eligible-users'='object|education_eligible_users'
        'GET /api/admin/dashboard'='object|admin_dashboard'; 'GET /api/admin/users/filters'='object|admin_user_filters'; 'GET /api/admin/role-permissions/filters'='object|admin_role_permission_filters'; 'GET /api/admin/position-roles/filters'='object|admin_position_role_filters'; 'GET /api/admin/permissions/assignments/filters'='object|admin_permission_assignment_filters'; 'GET /api/admin/audit'='page|backend/internal/admin/models.go|AuditEvent'; 'GET /api/admin/audit/filters'='object|admin_audit_filters'; 'GET /api/admin/dossier-requirements/filters'='object|admin_dossier_filters'; 'GET /api/admin/workflow-definitions/filters'='object|admin_workflow_filters'; 'GET /api/admin/nomenclatures/filters'='object|admin_nomenclature_filters'; 'GET /api/admin/education-taxonomies/filters'='object|admin_education_taxonomy_filters'
        'GET /api/gdpr/dashboard'='dto|backend/internal/gdpr/models.go|DashboardResponse'; 'GET /api/gdpr/config'='dto|backend/internal/gdpr/models.go|ConfigResponse'; 'GET /api/gdpr/retention-policies/filters'='dto|backend/internal/gdpr/models.go|RetentionPolicyFiltersResponse'; 'GET /api/gdpr/subject-requests/filters'='dto|backend/internal/gdpr/models.go|SubjectRequestFiltersResponse'; 'GET /api/gdpr/exports/dashboard'='dto|backend/internal/gdpr/models.go|ExportDashboardResponse'; 'GET /api/gdpr/exports/filters'='dto|backend/internal/gdpr/models.go|SubjectExportFiltersResponse'; 'GET /api/gdpr/publication-reviews/dashboard'='dto|backend/internal/gdpr/models.go|PublicationDashboardResponse'; 'GET /api/gdpr/publication-reviews/filters'='dto|backend/internal/gdpr/models.go|PublicationReviewFiltersResponse'
    }
    if (-not $map.ContainsKey($operationKey)) { return $null }
    return $map[$operationKey].Split('|', 3)
}

function New-ExactInlineResponseSchema([string]$name) {
    $string=@{type='string'}; $boolean=@{type='boolean'}; $integer=@{type='integer'}; $strings=@{type='array';items=@{type='string'}}
    switch ($name) {
        'education_eligible_users' { return @{type='object';additionalProperties=$false;required=@('items');properties=@{items=@{type='array';items=@{type='object';additionalProperties=$false;required=@('id','name');properties=@{id=$string;name=$string}}}}} }
		'school_operations_contract_create' { return @{type='object';additionalProperties=$false;required=@('id','expected_version','archive_status');properties=@{id=@{type='string';format='uuid'};expected_version=$integer;archive_status=$string;idempotent=$boolean}} }
		'school_operations_contract_amend' { return @{type='object';additionalProperties=$false;required=@('id','expected_version');properties=@{id=@{type='string';format='uuid'};expected_version=$integer}} }
		'school_operations_contract_obligation_create' { return @{type='object';additionalProperties=$false;required=@('id','expected_version');properties=@{id=@{type='string';format='uuid'};expected_version=$integer}} }
		'school_operations_contract_transition' { return @{type='object';additionalProperties=$false;required=@('id','lifecycle_status','expected_version');properties=@{id=@{type='string';format='uuid'};lifecycle_status=$string;expected_version=$integer}} }
        'auth_methods' { return @{type='object';additionalProperties=$false;required=@('methods');properties=@{methods=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{code=$string;label=$string;enabled=$boolean;primary=$boolean}}}}} }
        'auth_ui_config' { return @{type='object';additionalProperties=$false;properties=@{auth_flow=$string;default_locale=$string;available_locales=$strings;theme_family=$string;theme_brand=$string;oidc_issuer=@{type='string';format='uri'};oidc_client_id=$string;desktop_client_id=$string;sms_otp_enabled=$boolean;passkey_enabled=$boolean;eudi_wallet_enabled=$boolean;gdpr_features_enabled=$boolean}} }
        'passkey_login_options' { return @{type='object';additionalProperties=$false;required=@('status','options');properties=@{status=$string;options=@{type='object';additionalProperties=$false;properties=@{challenge=$string;rpId=$string;timeout=$integer;userVerification=$string;allowCredentials=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{type=$string;id=$string}}}}}}} }
        'passkey_login_finish' { return @{type='object';additionalProperties=$false;required=@('nonce');properties=@{nonce=$string}} }
        'passkey_register_options' { return @{type='object';additionalProperties=$false;properties=@{challenge=$string;rp=@{type='object';additionalProperties=$false;properties=@{name=$string;id=$string}};user=@{type='object';additionalProperties=$false;properties=@{id=$string;name=$string;displayName=$string}};pubKeyCredParams=@{type='array';items=@{type='object';additionalProperties=$false;properties=@{type=$string;alg=$integer}}};timeout=$integer;attestation=$string;authenticatorSelection=@{type='object';additionalProperties=$false;properties=@{residentKey=$string;requireResidentKey=$boolean;userVerification=$string}}}} }
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

function Get-EducationHandlerCreatedStatus([hashtable]$coverageOperation) {
    if (-not $coverageOperation -or [string]$coverageOperation.operationKey -notlike 'POST *' -or -not $coverageOperation.handler -or -not $coverageOperation.source) { return $null }
    $sourcePath = ([string]$coverageOperation.source) -replace ':\d+$', ''
    if (-not (Test-Path $sourcePath)) { return $null }
    $handler = [regex]::Escape([string]$coverageOperation.handler)
    if (-not $script:educationHandlerSources) { $script:educationHandlerSources = @{} }
    if (-not $script:educationHandlerSources.ContainsKey($sourcePath)) { $script:educationHandlerSources[$sourcePath] = Get-Content -Raw $sourcePath }
    $source = $script:educationHandlerSources[$sourcePath]
    $functionMatch = [regex]::Match($source, "(?ms)^func\s+\(s\s+\*Service\)\s+$handler\s*\(.*?(?=^func\s|\z)")
    if ($functionMatch.Success -and $functionMatch.Value -match 'http\.StatusCreated') { return '201' }
    return $null
}

function New-QueryParameter([string]$operationKey, [string]$parameterName) {
    if ($operationKey -eq 'GET /api/education/portfolios/records/{recordID}/lifecycle-operations') {
        if ($parameterName -eq 'page') { return [ordered]@{ name = 'page'; in = 'query'; required = $false; schema = @{ type = 'integer'; minimum = 1; maximum = 1000000; default = 1 } } }
        if ($parameterName -eq 'pageSize') { return [ordered]@{ '$ref' = '#/components/parameters/PageSize' } }
        $schema = [ordered]@{ type='string' }
        if ($parameterName -eq 'sort') { $schema.enum=@('requested_at','status','type'); $schema.default='requested_at' }
        if ($parameterName -eq 'direction') { $schema.enum=@('asc','desc'); $schema.default='desc' }
        if ($parameterName -eq 'filter.status') { $schema.enum=@('pending','processing','completed','blocked','dead_letter') }
        if ($parameterName -eq 'filter.type') { $schema.enum=@('cessation_retention','legal_hold_reconcile') }
        if ($parameterName -eq 'filter.requested_at') { $schema.format='date' }
        return [ordered]@{ name=$parameterName; in='query'; required=$false; schema=$schema }
    }
    if ($operationKey -eq 'GET /api/regulatory-sources') {
        if ($parameterName -eq 'page') { return [ordered]@{ '$ref' = '#/components/parameters/Page' } }
        if ($parameterName -eq 'pageSize') { return [ordered]@{ '$ref' = '#/components/parameters/PageSize' } }
        $schema = [ordered]@{ type='string' }
        if ($parameterName -eq 'status') { $schema.enum=@('draft','verified','active','superseded','withdrawn') }
        if ($parameterName -eq 'source_kind') { $schema.enum=@('law','government_decision','ministerial_order','authorization','accreditation','founder_decision','contract','other') }
        if ($parameterName -eq 'direction') { $schema.enum=@('asc','desc') }
        if ($parameterName -eq 'sort') { $schema.enum=@('citation','source_kind','status','created_at','updated_at') }
        return [ordered]@{ name=$parameterName; in='query'; required=$false; schema=$schema }
    }
    if ($operationKey -eq 'GET /api/earchiva/retention-rules') {
        if ($parameterName -eq 'page') { return [ordered]@{ '$ref' = '#/components/parameters/Page' } }
        if ($parameterName -eq 'pageSize') { return [ordered]@{ '$ref' = '#/components/parameters/PageSize' } }
        $schema = [ordered]@{ type='string' }
        if ($parameterName -in @('taxonomy_node_id','source_id')) { $schema.format='uuid' }
        if ($parameterName -eq 'status') { $schema.enum=@('proposed','active','retired','revoked') }
        if ($parameterName -eq 'anchor_kind') { $schema.enum=@('intake_received_at','event','permanent') }
        if ($parameterName -eq 'direction') { $schema.enum=@('asc','desc') }
        if ($parameterName -eq 'sort') { $schema.enum=@('effective_from','effective_to','status','minimum_retention_days','created_at','updated_at') }
        return [ordered]@{ name=$parameterName; in='query'; required=$false; schema=$schema }
    }
    if ($operationKey -eq 'GET /api/earchiva/admin/portfolio-custody-intents') {
        if ($parameterName -eq 'page') { return [ordered]@{ '$ref' = '#/components/parameters/Page' } }
        if ($parameterName -eq 'pageSize') { return [ordered]@{ '$ref' = '#/components/parameters/PageSize' } }
        $schema = [ordered]@{ type='string' }
        if ($parameterName -in @('intent_id','portfolio_id')) { $schema.format='uuid' }
        if ($parameterName -eq 'status') { $schema.enum=@('stored','queued','leased','committed','blocked','deadletter') }
        if ($parameterName -eq 'disposition') { $schema.enum=@('teacher_access','institution_archive_only') }
        if ($parameterName -in @('created_from','created_to')) { $schema.format='date-time' }
        if ($parameterName -eq 'direction') { $schema.enum=@('asc','desc'); $schema.default='desc' }
        if ($parameterName -eq 'sort') { $schema.enum=@('created_at','intent_id','portfolio_id','status','disposition','title','original_file_name'); $schema.default='created_at' }
        return [ordered]@{ name=$parameterName; in='query'; required=$false; schema=$schema }
    }
    # Keep the generated client honest: these are the exact sort allowlists
    # passed to httpx.ParsePageQuery by the governance/managerial handlers.
    $educationListSorts = @{
		'GET /api/education/classes' = @('class_code', 'class_name', 'school_year', 'grade_level', 'active')
		'GET /api/education/students' = @('student_code', 'first_name', 'last_name', 'status', 'birth_date')
		'GET /api/education/class-enrolments' = @('student_name', 'class_name', 'enrolled_from', 'enrolled_until', 'status')
		'GET /api/education/homeroom-assignments' = @('teacher_name', 'class_name', 'assigned_from', 'assigned_until')
        'GET /api/education/governance/memberships' = @('school_year', 'organism', 'full_name', 'role_name', 'status')
        'GET /api/education/governance/bodies' = @('school_year', 'organism', 'active_members', 'voting_members', 'held_meetings', 'latest_meeting_on')
        'GET /api/education/governance/meetings/{meetingID}/participants' = @('full_name', 'role_name', 'member_type', 'attendance_status', 'signature_present', 'voting_right')
        'GET /api/education/governance/meetings/{meetingID}/documents' = @('document_type', 'title', 'document_number', 'registry_number', 'publication_status', 'issued_on', 'custody_owner')
        'GET /api/education/governance/meetings/{meetingID}/votes' = @('subject_title', 'agenda_order', 'decision_type', 'outcome', 'requires_follow_up')
        'GET /api/education/governance/meetings/{meetingID}/minutes' = @('agenda_order', 'topic_title', 'discussion_summary', 'decision_summary', 'follow_up_status', 'responsible_party', 'due_on', 'requires_publication', 'notes')
        'GET /api/education/governance/meetings/{meetingID}/resolutions' = @('resolution_code', 'title', 'resolution_type', 'publication_status', 'anonymization_state')
        'GET /api/education/decisions/records/{decisionID}/issuances' = @('issuance_code', 'document_type', 'recipient_name', 'recipient_role', 'delivery_channel', 'delivery_status')
        'GET /api/education/decisions/records/{decisionID}/publication-steps' = @('step_order', 'step_type', 'status', 'responsible_name', 'publication_channel', 'due_on', 'completed_on')
        'GET /api/education/regulations/records/{recordID}/versions' = @('version_label', 'version_status', 'prepared_by', 'approved_on', 'effective_from', 'published_on')
        'GET /api/education/regulations/records/{recordID}/workflow' = @('phase_order', 'phase_type', 'status', 'audience', 'started_on', 'due_on', 'completed_on', 'feedback_count')
        'GET /api/education/committees/records/{recordID}/members' = @('full_name', 'role_name', 'member_type', 'status', 'appointed_on')
        'GET /api/education/managerial/records/{recordID}/documents' = @('document_code', 'document_category', 'title', 'document_status', 'version_label', 'owner_name')
        'GET /api/education/managerial/records/{recordID}/workflow' = @('stage_order', 'stage_type', 'status', 'assigned_to', 'due_on', 'completed_on')
    }
    if ($educationListSorts.ContainsKey($operationKey)) {
        if ($parameterName -eq 'page') { return [ordered]@{ '$ref' = '#/components/parameters/Page' } }
        if ($parameterName -eq 'pageSize') { return [ordered]@{ '$ref' = '#/components/parameters/PageSize' } }
        $schema = if ($parameterName -eq 'sort') { [ordered]@{ type = 'string'; enum = $educationListSorts[$operationKey] } } elseif ($parameterName -eq 'direction') { [ordered]@{ type = 'string'; enum = @('asc', 'desc') } } else { [ordered]@{ type = 'string' } }
        return [ordered]@{ name = $parameterName; in = 'query'; required = $false; schema = $schema }
    }
    $portfolioRelationSorts = @{
        'GET /api/education/portfolios/records/{recordID}/documents' = @('section_code', 'component_code', 'document_title', 'description', 'school_year', 'subject_discipline', 'applicable_class', 'source_scope', 'evidence_type', 'issued_on', 'chronological_index', 'sensitive_data', 'authenticity_status', 'archive_version_no')
		'GET /api/education/portfolios/me/{recordID}/documents' = @('section_code', 'component_code', 'document_title', 'description', 'school_year', 'subject_discipline', 'applicable_class', 'source_scope', 'evidence_type', 'issued_on', 'chronological_index', 'sensitive_data', 'authenticity_status', 'archive_version_no')
		'GET /api/education/portfolios/records/{recordID}/documents/{documentID}/versions' = @('version_no', 'change_type', 'changed_by', 'changed_at', 'reason')
		'GET /api/education/portfolios/me/{recordID}/documents/{documentID}/versions' = @('version_no', 'change_type', 'changed_by', 'changed_at', 'reason')
        'GET /api/education/portfolios/records/{recordID}/checklist' = @('requirement_code', 'requirement_label', 'section_code', 'status', 'document_count')
        'GET /api/education/portfolios/records/{recordID}/opis' = @('section_code', 'component_code', 'entry_title', 'chronological_index', 'document_reference')
        'GET /api/education/portfolios/records/{recordID}/custody' = @('event_type', 'holder_name', 'holder_role', 'started_on', 'ended_on')
        'GET /api/education/portfolios/records/{recordID}/reviews' = @('review_code', 'review_stage', 'outcome', 'reviewer_name', 'reviewed_on')
    }
    if ($portfolioRelationSorts.ContainsKey($operationKey)) {
        if ($parameterName -eq 'page') { return [ordered]@{ '$ref' = '#/components/parameters/Page' } }
        if ($parameterName -eq 'pageSize') { return [ordered]@{ '$ref' = '#/components/parameters/PageSize' } }
        $schema = if ($parameterName -eq 'sort') {
            [ordered]@{ type = 'string'; enum = $portfolioRelationSorts[$operationKey] }
        } elseif ($parameterName -eq 'direction') {
            [ordered]@{ type = 'string'; enum = @('asc', 'desc') }
        } else {
            [ordered]@{ type = 'string' }
        }
        return [ordered]@{ name = $parameterName; in = 'query'; required = $false; schema = $schema }
    }

    if ($operationKey -eq 'GET /api/education/signatures/eligible-artifacts') {
        if ($parameterName -eq 'page') { return [ordered]@{ '$ref' = '#/components/parameters/Page' } }
        if ($parameterName -eq 'pageSize') { return [ordered]@{ '$ref' = '#/components/parameters/PageSize' } }
        $schema = if ($parameterName -eq 'artifactType') {
            [ordered]@{ type = 'string'; enum = @('decision', 'publication', 'managerial_document', 'meeting_document', 'meeting_minute', 'meeting_resolution') }
        } else { [ordered]@{ type = 'string' } }
        return [ordered]@{ name = $parameterName; in = 'query'; required = ($parameterName -eq 'artifactType'); schema = $schema }
    }

    if ($operationKey -eq 'GET /api/education/classes/assignment-options') {
        if ($parameterName -eq 'page') { return [ordered]@{ '$ref' = '#/components/parameters/Page' } }
        if ($parameterName -eq 'pageSize') { return [ordered]@{ '$ref' = '#/components/parameters/PageSize' } }
        $schema = if ($parameterName -eq 'kind') {
            [ordered]@{ type = 'string'; enum = @('classes', 'students', 'teachers') }
        } else {
            [ordered]@{ type = 'string' }
        }
        return [ordered]@{ name = $parameterName; in = 'query'; required = ($parameterName -eq 'kind'); schema = $schema }
    }

    if ($operationKey -eq 'GET /api/education/portfolios/records/{recordID}/valorification-packages') {
        $schema = if ($parameterName -eq 'sort') {
            [ordered]@{ type = 'string'; enum = @('created_at', 'scope', 'purpose', 'status') }
        } elseif ($parameterName -eq 'direction') {
            [ordered]@{ type = 'string'; enum = @('asc', 'desc') }
        } else {
            [ordered]@{ type = 'string' }
        }
        if ($parameterName -eq 'page') { return [ordered]@{ '$ref' = '#/components/parameters/Page' } }
        if ($parameterName -eq 'pageSize') { return [ordered]@{ '$ref' = '#/components/parameters/PageSize' } }
        return [ordered]@{ name = $parameterName; in = 'query'; required = $false; schema = $schema }
    }

    if ($operationKey -eq 'GET /api/registratura/documents') {
        $pagination = [ordered]@{ type = 'integer'; minimum = 1; maximum = 100; default = 25 }
        $sortFields = @('registry_number', 'external_number', 'subject', 'document_type', 'direction', 'status', 'correspondent', 'assigned_to', 'confidentiality', 'registered_at', 'entry_at', 'exit_at')
        $dateFilters = @('filter.registered_at', 'filter.registered_at_from', 'filter.registered_at_to', 'filter.entry_at_from', 'filter.entry_at_to', 'filter.exit_at_from', 'filter.exit_at_to', 'filter.due_date', 'filter.due_date_from', 'filter.due_date_to')
        $schema = if ($parameterName -eq 'page') {
            [ordered]@{ type = 'integer'; minimum = 1; default = 1 }
        } elseif ($parameterName -in @('pageSize', 'limit')) {
            $pagination
        } elseif ($parameterName -in @('sort', 'sortBy')) {
            [ordered]@{ type = 'string'; enum = $sortFields }
        } elseif ($parameterName -in @('direction', 'sortDir')) {
            [ordered]@{ type = 'string'; enum = @('asc', 'desc') }
        } elseif ($parameterName -eq 'filter.registru_id') {
            [ordered]@{ type = 'integer'; format = 'int64'; minimum = 1 }
        } elseif ($parameterName -in $dateFilters) {
            [ordered]@{ type = 'string'; format = 'date' }
        } else {
            [ordered]@{ type = 'string' }
        }
        return [ordered]@{ name = $parameterName; in = 'query'; required = $false; schema = $schema }
    }

	$schoolOperationsSorts = @{
		'GET /api/school-operations/suppliers' = @('display_name','code','tax_id')
		'GET /api/school-operations/contracts' = @('contract_number','supplier_name','title','category','lifecycle_status','starts_on','ends_on','total_value','archive_status')
		'GET /api/school-operations/contracts/{contractID}/obligations' = @('title','status','due_on')
	}
	if ($schoolOperationsSorts.ContainsKey($operationKey)) {
		if ($parameterName -eq 'page') { return [ordered]@{ '$ref' = '#/components/parameters/Page' } }
		if ($parameterName -eq 'pageSize') { return [ordered]@{ '$ref' = '#/components/parameters/PageSize' } }
		$schema = if ($parameterName -eq 'sort') {
			[ordered]@{ type = 'string'; enum = $schoolOperationsSorts[$operationKey] }
		} elseif ($parameterName -eq 'direction') {
			[ordered]@{ type = 'string'; enum = @('asc','desc') }
		} elseif ($parameterName -in @('filter.starts_on','filter.ends_on','filter.due_on')) {
			[ordered]@{ type = 'string'; format = 'date' }
		} else {
			[ordered]@{ type = 'string' }
		}
		return [ordered]@{ name = $parameterName; in = 'query'; required = $false; schema = $schema }
	}

    if ($parameterName -eq 'page') { return [ordered]@{ '$ref' = '#/components/parameters/Page' } }
    if ($parameterName -eq 'pageSize') { return [ordered]@{ '$ref' = '#/components/parameters/PageSize' } }
    return [ordered]@{ name = $parameterName; in = 'query'; required = $false; schema = [ordered]@{ type = 'string' } }
}

$routerSource = Get-Content -Raw $Router
$common = Get-Content -Raw 'openapi/components/common.json' | ConvertFrom-Json -AsHashtable
$common.components.schemas['OwnPortfolioAppliedProcedureResponse'] = Get-GoDTOObjectSchema 'backend/internal/education/portfolio_own_procedure.go|OwnPortfolioAppliedProcedureResponse'
# These DTOs are declared in another Go source file. Reuse their canonical
# schemas instead of accepting the source-local helper's unresolved string fallback.
$common.components.schemas.OwnPortfolioAppliedProcedureResponse.properties.procedure = [ordered]@{ '$ref' = '#/components/schemas/PortfolioProcedure' }
$common.components.schemas.OwnPortfolioAppliedProcedureResponse.properties.rules.items = [ordered]@{ '$ref' = '#/components/schemas/PortfolioProcedureSectionRule' }
$common.components.schemas['AdmissionDSSRetentionPolicy'] = Get-GoDTOObjectSchema 'backend/internal/admission/models.go|AdmissionDSSRetentionPolicy'
$common.components.schemas['AdmissionRetentionRuleVersion'] = Get-GoDTOObjectSchema 'backend/internal/admission/models.go|AdmissionRetentionRuleVersion'
$common.components.schemas['ConfigureDSSRetentionPolicyRequest'] = Get-GoDTOObjectSchema 'backend/internal/admission/dss_retention_policy.go|ConfigureDSSRetentionPolicyRequest'
$overrides = Get-Content -Raw 'openapi/overrides.json' | ConvertFrom-Json -AsHashtable
$domainRules = @()
$common.components.schemas['PortfolioLifecycleOperation'] = Get-GoDTOObjectSchema 'backend/internal/education/portfolio_models.go|PortfolioLifecycleOperation'
$common.components.schemas['PortfolioLifecycleRetryRequest'] = Get-GoDTOObjectSchema 'backend/internal/education/portfolio_models.go|PortfolioLifecycleRetryRequest'
$common.components.schemas.PortfolioLifecycleRetryRequest.properties.reason.minLength = 1
$common.components.schemas.PortfolioLifecycleOperation.properties.status.enum = @('pending','processing','completed','blocked','dead_letter')
$common.components.schemas.PortfolioLifecycleOperation.properties.type.enum = @('cessation_retention','legal_hold_reconcile')
foreach ($countField in @('total_versions','completed_versions','blocked_versions')) { $common.components.schemas.PortfolioLifecycleOperation.properties[$countField].minimum = 0 }
$common.components.schemas['PortfolioLifecycleOperationResponse'] = Get-GoDTOObjectSchema 'backend/internal/education/portfolio_models.go|PortfolioLifecycleOperationResponse'
$common.components.schemas.PortfolioLifecycleOperationResponse.properties.operation = @{ '$ref' = '#/components/schemas/PortfolioLifecycleOperation' }
$common.components.schemas.PortfolioLifecycleOperationResponse.properties.portfolio = @{ '$ref' = '#/components/schemas/PortfolioRecord' }
$common.components.schemas['PortfolioStorageTransition'] = Get-GoDTOObjectSchema 'backend/internal/education/portfolio_models.go|PortfolioStorageTransition'
$common.components.schemas.PortfolioStorageTransition.properties.status.enum = @('pending','processing','completed','blocked','dead_letter')
$common.components.schemas.PortfolioLifecycleOperationResponse.properties.transitions = @{ type='array'; items=@{ '$ref' = '#/components/schemas/PortfolioStorageTransition' } }
$common.components.schemas['PortfolioRetentionDisposition'] = Get-GoDTOObjectSchema 'backend/internal/education/portfolio_models.go|PortfolioRetentionDisposition'
$common.components.schemas['PortfolioRetentionDispositionEvidence'] = Get-GoDTOObjectSchema 'backend/internal/education/portfolio_models.go|PortfolioRetentionDispositionEvidence'
$common.components.schemas['PortfolioRetentionDispositionRequest'] = Get-GoDTOObjectSchema 'backend/internal/education/portfolio_models.go|PortfolioRetentionDispositionRequest'
$common.components.schemas['PortfolioRetentionDispositionDecisionRequest'] = Get-GoDTOObjectSchema 'backend/internal/education/portfolio_models.go|PortfolioRetentionDispositionDecisionRequest'
$common.components.schemas['PortfolioRetentionDispositionCommandResponse'] = Get-GoDTOObjectSchema 'backend/internal/education/portfolio_models.go|PortfolioRetentionDispositionCommandResponse'
$common.components.schemas.PortfolioRetentionDisposition.properties.status.enum = @('submitted','approved','rejected','blocked','closed')
$common.components.schemas.PortfolioRetentionDisposition.properties.decision.enum = @('approved','rejected')
$common.components.schemas.PortfolioRetentionDisposition.properties.outcome.enum = @('released','retained','blocked')
$common.components.schemas.PortfolioRetentionDisposition.properties.operation_status.enum = @('queued','leased','released','blocked','deadletter')
$common.components.schemas.PortfolioRetentionDisposition.properties.operation_attempts.minimum = 0
$common.components.schemas.PortfolioRetentionDisposition.properties.evidence = @{ '$ref' = '#/components/schemas/PortfolioRetentionDispositionEvidence' }
$common.components.schemas.PortfolioRetentionDispositionRequest.properties.evidence = @{ '$ref' = '#/components/schemas/PortfolioRetentionDispositionEvidence' }
$common.components.schemas.PortfolioRetentionDispositionEvidence.properties.statement.minLength = 1
$common.components.schemas.PortfolioRetentionDispositionEvidence.properties.statement.maxLength = 4000
$common.components.schemas.PortfolioRetentionDispositionEvidence.properties.reference.maxLength = 500
$common.components.schemas.PortfolioRetentionDispositionDecisionRequest.properties.reason.minLength = 1
$common.components.schemas.PortfolioRetentionDispositionDecisionRequest.properties.reason.maxLength = 2000
$common.components.schemas['EducationPageOfPortfolioLifecycleOperation'] = @{
    type='object'; additionalProperties=$false; required=@('items','page','pageSize','total')
    properties=@{
        items=@{type='array';items=@{'$ref'='#/components/schemas/PortfolioLifecycleOperation'}}
        page=@{type='integer';minimum=1}; pageSize=@{type='integer';minimum=1}; total=@{type='integer';minimum=0}
    }
}
$common.components.schemas['EducationPageOfPortfolioRetentionDisposition'] = @{
    type='object';additionalProperties=$false;required=@('items','page','pageSize','total');properties=@{
        items=@{type='array';items=@{'$ref'='#/components/schemas/PortfolioRetentionDisposition'}}
        page=@{type='integer';minimum=1};pageSize=@{type='integer';minimum=1};total=@{type='integer';minimum=0}
    }
}
$domainCoverage = @{}
$educationRequestSchemas = @{}

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
            if ($coveredOperation.operationKey -like '* /api/education/*' -and $coveredOperation.requestBody -and $coveredOperation.requestBody.schema) {
                $educationRequestSchemas[[string]$coveredOperation.requestBody.schema] = $true
            }
        }
    }
}

# Response requiredness is derived from the Go JSON contract, not maintained by
# hand in the generated catalogue. A field without `omitempty` is always emitted
# by encoding/json and is therefore required in response schemas. Request DTOs
# are excluded because their semantic requiredness comes from handler validation.
$educationRequestRequiredFields = @{
    'CreateGovernanceMeetingParticipantRequest' = @('full_name','role_name','member_type','attendance_status')
    'CreateGovernanceMeetingDocumentRequest' = @('document_type','title','publication_status','issued_on')
    'CreateGovernanceMeetingVoteRequest' = @('agenda_order','subject_title','decision_type','outcome')
    'CreateGovernanceMinuteItemRequest' = @('agenda_order','topic_title','discussion_summary','decision_summary','follow_up_status')
    'CreateGovernanceResolutionRequest' = @('vote_id','title','resolution_type','publication_status','anonymization_state','issued_on')
    'CreateDecisionIssuanceRequest' = @('document_type','recipient_name','delivery_channel','delivery_status')
    'CreateDecisionPublicationStepRequest' = @('step_order','step_type','status','responsible_name','due_on')
    'CreateRegulationVersionRequest' = @('version_label','version_status','change_summary','effective_from','prepared_by')
    'CreateRegulationWorkflowStepRequest' = @('phase_order','phase_type','status','audience','started_on','due_on')
    'CreateCommitteeMemberRequest' = @('full_name','role_name','member_type','status','appointed_on')
    'CreateManagerialDocumentRequest' = @('document_category','title','document_status','version_label','registered_on')
    'CreateManagerialWorkflowStepRequest' = @('stage_order','stage_type','status','assigned_to','due_on')
    'CreatePersonnelAssignmentRequest' = @('assignment_type','assignment_title','status','assigned_on')
    'CreatePersonnelPersonalFileDocumentRequest' = @('document_category','document_title','file_scope','confidentiality_level','issued_on')
    'CreatePersonnelDisciplinaryCaseRequest' = @('case_type','status','reported_on')
    'CreatePersonnelPersonalAccessEventRequest' = @('event_type','actor_name','actor_role','purpose','access_channel','accessed_on')
    'CreatePersonnelEvaluationSelfReviewRequest' = @('section_title','narrative_type','status','completed_on')
    'CreatePersonnelEvaluationCriterionRequest' = @('criterion_category','criterion_label','status','max_score')
    'CreatePersonnelEvaluationAppealRequest' = @('submitted_by','submitted_on','status','grounds')
    'CreatePersonnelEvaluationResultIssueRequest' = @('document_type','recipient_name','delivery_channel','delivery_status','issued_on')
    'CreateMobilityDocumentRequest' = @('document_type','stage_scope','document_title','registered_on','validation_status')
    'CreateMobilityCriterionScoreRequest' = @('criterion_code','criterion_label','criterion_category','max_score')
    'CreateMobilityAppealRequest' = @('submitted_by','submitted_on','status','grounds')
    'CreateMobilityFinalDecisionRequest' = @('decision_type','outcome','approved_on','effective_from','panel_name')
    'CreateMobilityResultIssueRequest' = @('document_type','recipient_name','delivery_channel','delivery_status','issued_on')
    'CreateMeritDocumentRequest' = @('document_type','document_title','registered_on','validation_status')
    'CreateMeritCriterionScoreRequest' = @('criterion_code','criterion_label','criterion_category','panel_stage','max_score')
    'CreateMeritAppealRequest' = @('submitted_by','submitted_on','status','grounds')
    'CreateMeritFinalDecisionRequest' = @('decision_stage','outcome','approved_on','effective_from','panel_name')
    'CreateMeritResultIssueRequest' = @('document_type','recipient_name','delivery_channel','delivery_status','issued_on')
    'CreatePortfolioDocumentRequest' = @('section_code','component_code','document_title','description','school_year','subject_discipline','applicable_class','competencies','source_scope','evidence_type','issued_on','added_on','authenticity_status','file_reference')
    'OwnPortfolioDocumentRequest' = @('section_code','component_code','document_title','description','school_year','subject_discipline','applicable_class','competencies','evidence_type','issued_on','added_on','file_reference')
    'CreatePortfolioChecklistItemRequest' = @('requirement_code','requirement_label','section_code','source_scope','status','last_checked_on')
    'CreatePortfolioOpisEntryRequest' = @('section_code','component_code','entry_title','source_scope','document_reference','checked_on')
	'CreatePortfolioCustodyEventRequest' = @('event_type','holder_name','holder_role','location_label','access_reason','started_on','access_mode')
	'PortfolioRetentionDispositionRequest' = @('evidence')
	'PortfolioRetentionDispositionDecisionRequest' = @('approve','reason')
    'CreatePortfolioReviewEventRequest' = @('review_stage','outcome','reviewed_on')
    'CreateClassOfferingContextRequest' = @('class_id','offering_id','location_id','authorization_id','school_year','shift','effective_from')
    'CreateCampaignRequest' = @('source_id','code','title','school_year','offering_id','location_id','authorization_id','class_offering_context_id','capacity_limit','capacity_unit','student_place_limit','capacity_basis','shift','opens_on','closes_on','criteria','document_requirements')
    'TransitionRequest' = @('status','expected_version')
    'CriterionInput' = @('code','title','kind','required','weight','ordinal','rule_snapshot')
    'DocumentRequirementInput' = @('code','title','required','allowed_mime_types','ordinal')
    'CreateApplicationRequest' = @('campaign_id','application_no','candidate_party_id','consent_snapshot')
    'AssessCriterionRequest' = @('criterion_id','outcome','rationale','evidence_snapshot','expected_version')
    'ReviewApplicationDocumentRequest' = @('status','review_note','expected_version')
    'IssueDecisionRequest' = @('decision_no','outcome','rationale','expected_version','archive')
    'PrepareDecisionRequest' = @('decision_no','outcome','rationale','expected_version')
    'PrepareAppealResolutionRequest' = @('outcome','rationale','expected_version','application_expected_version')
    'FinalizeAdmissionLegalPreparationRequest' = @('preparation_id','archive')
    'ProposeAdmissionSignerAuthorizationRequest' = @('certificate_sha256','user_id','permission_code','valid_until')
    'ApproveAdmissionSignerAuthorizationRequest' = @('proposal_id')
    'RevokeAdmissionSignerAuthorizationRequest' = @('expected_version','reason')
    'CreateAppealRequest' = @('decision_id','appeal_no','submitted_by_party_id','statement')
    'EnrolApplicationRequest' = @('student_code','enrolled_from','expected_version')
    'ResolveAppealRequest' = @('outcome','rationale','expected_version','application_expected_version','resulting_decision_no','resulting_outcome','archive')
}
$educationSources = @(Get-ChildItem 'backend/internal/education/*.go' | Where-Object { $_.Name -notlike '*_test.go' })
foreach ($schemaName in @($common.components.schemas.Keys)) {
    $schema = $common.components.schemas[$schemaName]
    if (-not $schema -or -not $schema.Contains('x-go-model') -or -not $schema.Contains('properties')) { continue }
    $schema.additionalProperties = $false
    if ($educationRequestSchemas.ContainsKey([string]$schemaName)) {
        if ($educationRequestRequiredFields.ContainsKey([string]$schemaName)) {
            $required = @($educationRequestRequiredFields[[string]$schemaName])
            $unknownRequired = @($required | Where-Object { -not $schema.properties.Contains($_) })
            if ($unknownRequired.Count -gt 0) { throw "Education request required-field catalogue names unknown properties on ${schemaName}: $($unknownRequired -join ', ')" }
            $schema.required = $required
            [void]$schema.Remove('x-requiredness')
        }
        continue
    }
    $modelName = [string]$schema['x-go-model']
    if ([string]::IsNullOrWhiteSpace($modelName)) { continue }
    $declarations = @($educationSources | Select-String -Pattern ("^type\s+{0}\s+struct\s*\{{" -f [regex]::Escape($modelName)))
    if ($declarations.Count -ne 1) { continue }
    $modelSource = Get-Content -Raw $declarations[0].Path
    $modelBlock = [regex]::Match($modelSource, ("(?ms)^type\s+{0}\s+struct\s*\{{(.*?)^\}}" -f [regex]::Escape($modelName)))
    if (-not $modelBlock.Success) { continue }
    $required = @([regex]::Matches($modelBlock.Groups[1].Value, 'json:"([^",]+)([^"]*)"') |
        Where-Object { $_.Groups[1].Value -ne '-' -and $_.Groups[2].Value -notmatch 'omitempty' -and $schema.properties.Contains($_.Groups[1].Value) } |
        ForEach-Object { $_.Groups[1].Value })
    if ($required.Count -gt 0) { $schema.required = $required }
    [void]$schema.Remove('x-requiredness')
}

# Handler-enforced finite values and conditional requirements are part of the
# public database-to-frontend contract, not knowledge that callers should have
# to recover from a late 400 response.
$committeeMemberSchema = $common.components.schemas['CreateCommitteeMemberRequest']
if ($committeeMemberSchema) {
    $committeeMemberSchema.properties.member_type.enum = @('presedinte','secretar','membru','observator','invitat')
    $committeeMemberSchema.properties.status.enum = @('active','inactive','replaced')
    $committeeMemberSchema.properties.appointed_on.format = 'date'
    $committeeMemberSchema.properties.released_on.format = 'date'
}
$managerialDocumentSchema = $common.components.schemas['CreateManagerialDocumentRequest']
if ($managerialDocumentSchema) {
    $managerialDocumentSchema.properties.document_category.enum = @('diagnoza','prognoza','evidenta','planificare','raport','anexa','hotarare','procedura')
    $managerialDocumentSchema.properties.document_status.enum = @('draft','in_review','approved','published','archived')
    $managerialDocumentSchema.properties.registered_on.format = 'date'
    $managerialDocumentSchema.properties.registered_on.minLength = 1
    $managerialDocumentSchema.properties.approved_on.format = 'date'
    $managerialDocumentSchema.properties.approved_on.minLength = 1
    $managerialDocumentSchema.allOf = @(
        [ordered]@{
            'if' = [ordered]@{ properties = [ordered]@{ document_status = [ordered]@{ enum = @('approved','published','archived') } }; required = @('document_status') }
            then = [ordered]@{ required = @('approved_on') }
        }
    )
}
$personnelRecordSchema = $common.components.schemas['CreatePersonnelRecordRequest']
if ($personnelRecordSchema) {
    $personnelRecordSchema.properties.employment_type.enum = @('titular','suplinitor','plata_cu_ora','auxiliar')
    $personnelRecordSchema.properties.status.enum = @('active','on_leave','vacant','inactive')
    $personnelRecordSchema.properties.evaluation_status.enum = @('draft','in_review','finalized')
    $personnelRecordSchema.properties.mobility_stage.enum = @('none','transfer','detasare','restrangere')
}
$publicationRecordSchema = $common.components.schemas['CreatePublicationRecordRequest']
if ($publicationRecordSchema) {
    $publicationRecordSchema.properties.domain.enum = @('guvernanta','documente_manageriale','portofolii','regulamente','conformitate')
    $publicationRecordSchema.properties.entity_type.enum = @('hotarare','proces_verbal','procedura_portofoliu','rof','roi','pdi_pas','raport','anunt')
    $publicationRecordSchema.properties.publication_channel.enum = @('site_public','avizier','intranet','registratura')
    $publicationRecordSchema.properties.publication_status.enum = @('pregatit','publicat','retras')
    $publicationRecordSchema.properties.anonymization_status.enum = @('necesara','finalizata','nu_este_necesara')
    $publicationRecordSchema.properties.published_on.format = 'date'
    $publicationRecordSchema.allOf = @([ordered]@{
        'if' = [ordered]@{ properties = [ordered]@{ publication_status = [ordered]@{ enum = @('publicat') } }; required = @('publication_status') }
        then = [ordered]@{ required = @('published_on') }
    })
}
$personnelAssignmentSchema = $common.components.schemas['CreatePersonnelAssignmentRequest']
if ($personnelAssignmentSchema) {
    $personnelAssignmentSchema.properties.assignment_type.enum = @('diriginte','coordonator_proiect','responsabil_comisie','mentor','membru_comisie','administrator_structura')
    $personnelAssignmentSchema.properties.status.enum = @('propus','activ','suspendat','incetat')
    $personnelAssignmentSchema.properties.assigned_on.format = 'date'
    $personnelAssignmentSchema.properties.ended_on.format = 'date'
    $personnelAssignmentSchema.properties.weekly_hours.minimum = 0
}
$personnelFileDocumentSchema = $common.components.schemas['CreatePersonnelPersonalFileDocumentRequest']
if ($personnelFileDocumentSchema) {
    $personnelFileDocumentSchema.properties.document_category.enum = @('identificare','studii','cariera','evaluare','declaratie','medical','disciplina','management')
    $personnelFileDocumentSchema.properties.file_scope.enum = @('dosar_personal','dosar_director','dosar_director_adjunct')
    $personnelFileDocumentSchema.properties.confidentiality_level.enum = @('intern','confidential','strict_confidential')
    $personnelFileDocumentSchema.properties.issued_on.format = 'date'
    $personnelFileDocumentSchema.properties.expires_on.format = 'date'
}
$personnelDisciplinarySchema = $common.components.schemas['CreatePersonnelDisciplinaryCaseRequest']
if ($personnelDisciplinarySchema) {
    $personnelDisciplinarySchema.properties.case_type.enum = @('sesizare','cercetare','sanctiune','contestatie')
    $personnelDisciplinarySchema.properties.status.enum = @('deschis','in_cercetare','solutionat','contestat','inchis')
    $personnelDisciplinarySchema.properties.reported_on.format = 'date'
    $personnelDisciplinarySchema.properties.hearing_on.format = 'date'
    $personnelDisciplinarySchema.properties.resolved_on.format = 'date'
}
$personnelAccessSchema = $common.components.schemas['CreatePersonnelPersonalAccessEventRequest']
if ($personnelAccessSchema) {
    $personnelAccessSchema.properties.event_type.enum = @('consultare','predare','actualizare','arhivare','export')
    $personnelAccessSchema.properties.access_channel.enum = @('fizic','digital','mixt')
    $personnelAccessSchema.properties.accessed_on.format = 'date'
    $personnelAccessSchema.properties.closed_on.format = 'date'
}
$evaluationSelfReviewSchema = $common.components.schemas['CreatePersonnelEvaluationSelfReviewRequest']
if ($evaluationSelfReviewSchema) {
    $evaluationSelfReviewSchema.properties.narrative_type.enum = @('autoevaluare','performanta','dezvoltare','impact')
    $evaluationSelfReviewSchema.properties.status.enum = @('draft','submitted','validated','returned')
    $evaluationSelfReviewSchema.properties.completed_on.format = 'date'
    $evaluationSelfReviewSchema.properties.assumed_score.minimum = 0
    $evaluationSelfReviewSchema.properties.assumed_score.maximum = 100
}
$evaluationCriterionSchema = $common.components.schemas['CreatePersonnelEvaluationCriterionRequest']
if ($evaluationCriterionSchema) {
    $evaluationCriterionSchema.properties.criterion_category.enum = @('proiectare','predare','evaluare','management_clasa','dezvoltare','parteneriat')
    $evaluationCriterionSchema.properties.status.enum = @('draft','reviewed','validated','contested')
    foreach ($scoreField in @('max_score','self_score','reviewer_score','final_score')) {
        $evaluationCriterionSchema.properties[$scoreField].minimum = 0
        $evaluationCriterionSchema.properties[$scoreField].maximum = 100
    }
    $evaluationCriterionSchema['x-cross-field-constraints'] = @([ordered]@{
        rule = 'criterion_scores_lte_max_score'
        expression = 'self_score <= max_score && reviewer_score <= max_score && final_score <= max_score'
        message = 'Every supplied score must be less than or equal to max_score.'
    })
}
$evaluationAppealSchema = $common.components.schemas['CreatePersonnelEvaluationAppealRequest']
if ($evaluationAppealSchema) {
    $evaluationAppealSchema.properties.status.enum = @('submitted','review','accepted','rejected','resolved')
    $evaluationAppealSchema.properties.submitted_on.format = 'date'
    $evaluationAppealSchema.properties.hearing_on.format = 'date'
    $evaluationAppealSchema.properties.resolved_on.format = 'date'
    $evaluationAppealSchema.allOf = @(
        [ordered]@{
            'if' = [ordered]@{ properties = [ordered]@{ status = [ordered]@{ enum = @('accepted','rejected','resolved') } }; required = @('status') }
            then = [ordered]@{ required = @('resolved_on') }
        },
        [ordered]@{
            'if' = [ordered]@{ properties = [ordered]@{ status = [ordered]@{ enum = @('accepted','rejected') } }; required = @('status') }
            then = [ordered]@{ required = @('decision_summary') }
        }
    )
}
$evaluationResultIssueSchema = $common.components.schemas['CreatePersonnelEvaluationResultIssueRequest']
if ($evaluationResultIssueSchema) {
    $evaluationResultIssueSchema.properties.document_type.enum = @('fisa_evaluare','comunicare','decizie','raport_final')
    $evaluationResultIssueSchema.properties.delivery_channel.enum = @('registratura','email','intern','posta')
    $evaluationResultIssueSchema.properties.delivery_status.enum = @('pregatit','emis','transmis','confirmat')
    $evaluationResultIssueSchema.properties.issued_on.format = 'date'
    $evaluationResultIssueSchema.properties.delivered_on.format = 'date'
    $evaluationResultIssueSchema.properties.acknowledged_on.format = 'date'
    $evaluationResultIssueSchema.allOf = @(
        [ordered]@{
            'if' = [ordered]@{ properties = [ordered]@{ delivery_status = [ordered]@{ enum = @('transmis','confirmat') } }; required = @('delivery_status') }
            then = [ordered]@{ required = @('delivered_on') }
        },
        [ordered]@{
            'if' = [ordered]@{ properties = [ordered]@{ delivery_status = [ordered]@{ enum = @('confirmat') } }; required = @('delivery_status') }
            then = [ordered]@{ required = @('acknowledged_on') }
        }
    )
}
$meritScoreSchema = $common.components.schemas['CreateMeritCriterionScoreRequest']
if ($meritScoreSchema) {
    $meritScoreSchema.properties.criterion_category.enum = @('performanta','impact','dezvoltare','management','incluziune')
    $meritScoreSchema.properties.panel_stage.enum = @('autoevaluare','evaluare_comisie','validare_finala')
    $meritScoreSchema.properties.max_score.exclusiveMinimum = 0
    $meritScoreSchema.properties.awarded_score.minimum = 0
    $meritScoreSchema.properties.awarded_score.description = 'Score awarded by the panel. When supplied, it MUST be less than or equal to max_score.'
    $meritScoreSchema['x-cross-field-constraints'] = @(
        [ordered]@{
            rule = 'awarded_score_lte_max_score'
            expression = 'awarded_score <= max_score'
            message = 'awarded_score must not exceed max_score when supplied'
        }
    )
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
	# A coverage catalogue is itself a complete, handler-backed contract.  New
	# covered operations do not need a duplicate global override merely to carry
	# their concrete schema and authorization metadata into the generator.
	if ($coverage -and -not $override) { $override = [ordered]@{} }
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
		$override.status = [string]$coverage.contractStatus
        $override.requiredPermission = $coverage.requiredPermission
        $override.pathParameters = @($coverage.pathParameters)
        $override.queryParameters = @($coverage.queryParameters)
        $override.errors = @($coverage.errors)
        $override.responseStatus = [string]$coverage.success.status
        $createdStatus = Get-EducationHandlerCreatedStatus $coverage
        if ($createdStatus) { $override.responseStatus = $createdStatus }
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
                $coverageResponseSchema = switch ([string]$coverage.success.contentType) {
                    'text/csv' { 'BinaryCsv' }
                    'application/zip' { 'BinaryZip' }
                    default { 'BinaryPdf' }
                }
                if ($coverageResponseSchema -eq 'BinaryZip') {
                    $common.components.schemas['BinaryZip'] = @{ type = 'string'; format = 'binary' }
                }
            }
            $override.responseSchema = $coverageResponseSchema
        }
        if ($coverage.requestBody -and $coverage.requestBody.schema) { $override.requestSchema = [string]$coverage.requestBody.schema }
        if ($coverage.requestBody) { $override.requestContentType = [string]$coverage.requestBody.contentType }
        if ($coverage.Contains('requestBody') -and -not $coverage.requestBody) { $override.requestBody = $false }
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
        # A coverage entry is a reviewed, handler-backed contract. Do not retain
        # the old "pending endpoint-level schema review" wording for a route
        # whose request, response, RBAC and tenant scope are already explicit.
        description = if ($override -and $override.description) { [string]$override.description } elseif ($coverage -and $coverage.modelEvidence) { "Handler-backed Education contract. $([string]$coverage.modelEvidence)" } elseif ($isDetailedFamily) { "Tenant-scoped $family operation. The backend is authoritative for RBAC, resource visibility, transition state and validation." } else { 'Generated router contract. Request and response field detail is pending endpoint-level schema review.' }
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
        '409' = '#/components/responses/Conflict'; '422' = '#/components/responses/Validation'; '500' = '#/components/responses/ServerError'
    }
    $errorStatuses = if ($override -and $override.errors) { @($override.errors | ForEach-Object { [string]$_ }) } else { @('400','401','403','404','422','500') }
    foreach ($errorStatus in $errorStatuses) {
        if ($errorResponseMap.ContainsKey($errorStatus)) { $operation.responses[$errorStatus] = [ordered]@{ '$ref' = $errorResponseMap[$errorStatus] } }
        elseif ($errorStatus -in @('413', '503')) {
            $operation.responses[$errorStatus] = [ordered]@{
                description = $(if ($errorStatus -eq '413') { 'Export or request exceeds server limits' } else { 'Required service temporarily unavailable' })
                content = @{ 'application/json' = @{ schema = @{ '$ref' = '#/components/schemas/Problem' } } }
            }
        }
    }

    $requiresSecurity = -not $isPublic
    if ($override -and $override.security -eq 'none') { $requiresSecurity = $false }
    if ($requiresSecurity) {
        if ($override -and $override.security -eq 'refreshCookie') {
            $operation.security = @(@{ refreshCookie = @() })
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
    if ($override -and $override.requiredPolicyCapability) { $operation.'x-required-policy-capability' = [string]$override.requiredPolicyCapability }

    $parameters = @()
    $pathParameterNames = if ($override -and $override.pathParameters) { @($override.pathParameters) } else { @([regex]::Matches($path, '\{([^}]+)\}') | ForEach-Object { $_.Groups[1].Value }) }
    foreach ($parameterName in $pathParameterNames) {
        $parameters += [ordered]@{ name = [string]$parameterName; in = 'path'; required = $true; schema = [ordered]@{ type = 'string' } }
    }
    if ($path -eq '/api/admissions/legal-preparations/{preparationID}/artifacts/{artifactSlot}') {
        foreach ($parameter in $parameters) {
            if ($parameter.name -eq 'preparationID') { $parameter.schema.format = 'uuid' }
            if ($parameter.name -eq 'artifactSlot') { $parameter.schema.enum = @('primary','resulting_decision') }
        }
        if ($method -eq 'post') { $operation.responses['200'] = [ordered]@{ description = 'Idempotent replay of the committed artifact'; content = [ordered]@{ 'application/json' = [ordered]@{ schema = [ordered]@{ '$ref' = "#/components/schemas/$responseSchema" } } } } }
    }
    if ($override -and $override.institutionContext) { $parameters += [ordered]@{ '$ref' = '#/components/parameters/Institution' } }

    $queryParameterNames = if ($override -and $override.queryParameters) { @($override.queryParameters) } else { @() }
    foreach ($parameterName in $queryParameterNames) {
        $parameters += New-QueryParameter $key ([string]$parameterName)
    }
    if ($method -eq 'post' -and ($path.StartsWith('/api/admissions/') -or $path -eq '/api/earchiva/retention-rules' -or $path.StartsWith('/api/earchiva/retention-rules/') -or $path -eq '/api/regulatory-sources' -or $path.StartsWith('/api/regulatory-sources/'))) {
        $parameters += [ordered]@{
            name = 'Idempotency-Key'
            in = 'header'
            required = $true
            schema = [ordered]@{ type = 'string'; minLength = 1; maxLength = 200 }
            description = 'Caller-generated key used to replay the same command safely within the authenticated tenant, institution and actor scope.'
        }
    }
    if ($parameters.Count -gt 0) { $operation.parameters = $parameters }
    $hasRequestBody = -not ($override -and $override.Contains('requestBody') -and -not [bool]$override.requestBody)
    # Activation is an authenticated, bodyless command.  The handler does not
    # decode JSON, so documenting a required empty object would reject valid
    # clients and create a false SDK method signature.
    if ($key -eq 'POST /api/eudi-wallet/activate') { $hasRequestBody = $false }
    if ($method -in @('post', 'put', 'patch') -and $hasRequestBody) {
        $requestSchema = if ($override -and $override.requestSchema) { [string]$override.requestSchema } elseif ($isDetailedFamily) { "$family`Request" } else { 'Mutation' }
		if ($key -eq 'PUT /api/institution/regulatory-profile') {
			$requestSchema = 'PutRegulatoryProfileRequest'
			if (-not $common.components.schemas.Contains($requestSchema)) { $common.components.schemas[$requestSchema] = New-ClosedRequestSchema $key }
		}
		if ($key -in @('POST /api/education/portfolios/me', 'PATCH /api/education/portfolios/me/{recordID}', 'POST /api/education/portfolios/me/{recordID}/documents', 'PATCH /api/education/portfolios/me/{recordID}/documents/{documentID}') -and -not $common.components.schemas.Contains($requestSchema)) {
			$common.components.schemas[$requestSchema] = New-ClosedRequestSchema $key
		}
		# Newly introduced command DTOs live outside the composed domain schema
		# fragments. Materialize their closed Go-derived schemas here so operation
		# overrides cannot leave dangling component references.
		if ($key -in @(
			'POST /api/education/portfolios/me/{recordID}/archive-documents',
			'POST /api/education/portfolios/records/{recordID}/return',
			'POST /api/education/portfolios/records/{recordID}/managerial-decision',
			'POST /api/institution/locations',
			'PATCH /api/institution/locations/{locationID}',
			'POST /api/institution/education-offerings',
			'PATCH /api/institution/education-offerings/{offeringID}',
			'POST /api/institution/offering-authorizations',
			'POST /api/school-operations/contracts',
			'PATCH /api/school-operations/contracts/{contractID}',
			'POST /api/school-operations/contracts/{contractID}/obligations',
			'POST /api/school-operations/contracts/{contractID}/transition',
			'POST /api/admissions/campaigns',
			'POST /api/admissions/class-offering-contexts',
			'POST /api/admissions/campaigns/{campaignID}/transitions',
			'POST /api/admissions/campaigns/{campaignID}/criteria',
			'POST /api/admissions/campaigns/{campaignID}/document-requirements',
			'POST /api/admissions/applications',
			'POST /api/admissions/applications/{applicationID}/transitions',
			'POST /api/admissions/applications/{applicationID}/assessments',
			'POST /api/admissions/applications/{applicationID}/documents/{documentID}',
			'POST /api/admissions/retention-rule-versions',
			'POST /api/admissions/retention-rule-versions/approve',
			'POST /api/admissions/dss-retention-policies',
			'POST /api/admissions/applications/{applicationID}/decision-preparations',
			'POST /api/admissions/appeals/{appealID}/resolution-preparations',
			'POST /api/admissions/legal-preparations/finalize',
			'POST /api/admissions/signer-authorizations',
			'POST /api/admissions/signer-authorizations/approve',
			'POST /api/admissions/signer-authorizations/{authorizationID}/revoke',
			'POST /api/admissions/applications/{applicationID}/decisions',
			'POST /api/admissions/applications/{applicationID}/appeals',
			'POST /api/admissions/applications/{applicationID}/enrolment',
			'POST /api/admissions/appeals/{appealID}/resolution',
			'POST /api/earchiva/admin/portfolio-custody-intents/{intentID}/reconcile'
		) -and -not $common.components.schemas.Contains($requestSchema)) {
			$common.components.schemas[$requestSchema] = New-ClosedRequestSchema $key
		}
		if ($common.components.schemas.Contains($requestSchema) -and $educationRequestRequiredFields.ContainsKey($requestSchema)) {
			$required = @($educationRequestRequiredFields[$requestSchema])
			$unknownRequired = @($required | Where-Object { -not $common.components.schemas[$requestSchema].properties.Contains($_) })
			if ($unknownRequired.Count -gt 0) { throw "Education request required-field catalogue names unknown properties on ${requestSchema}: $($unknownRequired -join ', ')" }
			$common.components.schemas[$requestSchema].required = $required
			[void]$common.components.schemas[$requestSchema].Remove('x-requiredness')
		}
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

$regulatoryProfileRequest = $common.components.schemas['PutRegulatoryProfileRequest']
if ($regulatoryProfileRequest) {
	$regulatorySourceRequest = Get-GoDTOObjectSchema 'backend/internal/institution/models.go|RegulatorySourceRequest'
	$regulatorySourceRequest.additionalProperties = $false
	$regulatorySourceRequest.properties.source_kind.enum = @('law','government_decision','ministerial_order','authorization','accreditation','founder_decision','contract','other')
	$regulatorySourceRequest.properties.source_url.format = 'uri'
	$regulatorySourceRequest.properties.published_on.format = 'date'
	$regulatorySourceRequest.properties.consolidated_on.format = 'date'
	$regulatorySourceRequest.properties.checksum_sha256.pattern = '^[a-f0-9]{64}$'
	$regulatorySourceRequest.required = @('source_kind','citation','article_reference','issuer','source_url','checksum_sha256')
	$common.components.schemas['RegulatorySourceRequest'] = $regulatorySourceRequest
	$regulatoryProfileRequest.properties.source = [ordered]@{ '$ref' = '#/components/schemas/RegulatorySourceRequest' }
	$confessionalOverlayRequest = Get-GoDTOObjectSchema 'backend/internal/institution/models.go|ConfessionalOverlayRequest'
	$confessionalOverlayRequest.additionalProperties = $false
	$confessionalOverlayRequest.properties.cult_party_id.format = 'uuid'
	$confessionalOverlayRequest.required = @('cult_party_id','cult_code','protocol_reference')
	$common.components.schemas['ConfessionalOverlayRequest'] = $confessionalOverlayRequest
	$regulatoryProfileRequest.properties.confessional_overlay = [ordered]@{ '$ref' = '#/components/schemas/ConfessionalOverlayRequest' }
	$regulatoryProfileRequest.additionalProperties = $false
	$regulatoryProfileRequest.properties.expected_version.minimum = 1
	$regulatoryProfileRequest.properties.status.enum = @('draft','approved','active')
	$regulatoryProfileRequest.properties.school_legal_form.enum = @('public','private')
	$regulatoryProfileRequest.properties.authorization_status.enum = @('unknown','provisional','authorized','accredited','suspended','withdrawn')
	$regulatoryProfileRequest.properties.effective_from.format = 'date'
	$regulatoryProfileRequest.properties.effective_to.format = 'date'
	$regulatoryProfileRequest.required = @('expected_version','status','school_legal_form','regulatory_profile','authorization_status','authorized_levels','has_legal_personality','program_codes','effective_from','source')
}

$locationRequest = $common.components.schemas['CreateSchoolLocationRequest']
if ($locationRequest) {
	$locationRequest.additionalProperties = $false
	$locationRequest.properties.effective_from.format = 'date'
	$locationRequest.properties.effective_to.format = 'date'
	$locationRequest.properties.idempotency_key.minLength = 1
	$locationRequest.properties.code.pattern = '^[a-z0-9][a-z0-9._-]{0,63}$'
	$locationRequest.required = @('code','name','active','effective_from','idempotency_key')
}
$locationUpdateRequest = $common.components.schemas['UpdateSchoolLocationRequest']
if ($locationUpdateRequest) {
	$locationUpdateRequest.additionalProperties = $false
	$locationUpdateRequest.properties.expected_version.minimum = 1
	$locationUpdateRequest.properties.effective_to.format = 'date'
	$locationUpdateRequest.required = @('expected_version','name','address','active')
}
$offeringRequest = $common.components.schemas['CreateEducationOfferingRequest']
if ($offeringRequest) {
	$offeringRequest.additionalProperties = $false
	foreach ($field in @('code','education_level','language_code')) { $offeringRequest.properties[$field].pattern = '^[a-z0-9][a-z0-9._-]{0,63}$' }
	$offeringRequest.properties.specialization_code.pattern = '^$|^[a-z0-9][a-z0-9._-]{0,63}$'
	$offeringRequest.properties.effective_from.format = 'date'
	$offeringRequest.properties.effective_to.format = 'date'
	$offeringRequest.properties.idempotency_key.minLength = 1
	$offeringRequest.required = @('code','education_level','language_code','title','active','effective_from','idempotency_key')
}
$offeringUpdateRequest = $common.components.schemas['UpdateEducationOfferingRequest']
if ($offeringUpdateRequest) {
	$offeringUpdateRequest.additionalProperties = $false
	$offeringUpdateRequest.properties.expected_version.minimum = 1
	$offeringUpdateRequest.properties.effective_to.format = 'date'
	$offeringUpdateRequest.required = @('expected_version','title','active')
}
$authorizationRequest = $common.components.schemas['CreateOfferingAuthorizationRequest']
if ($authorizationRequest) {
	$authorizationRequest.additionalProperties = $false
	$authorizationRequest.properties.offering_id.format = 'uuid'
	$authorizationRequest.properties.location_id.format = 'uuid'
	$authorizationRequest.properties.replaces_authorization_id.format = 'uuid'
	$authorizationRequest.properties.expected_version.minimum = 1
	$authorizationRequest.properties.capacity.minimum = 0
	$authorizationRequest.properties.status.enum = @('provisional','accredited','suspended','withdrawn','expired')
	$authorizationRequest.properties.capacity_unit.enum = @('students','study_groups')
	$authorizationRequest.properties.shift.enum = @('day','afternoon','evening')
	$authorizationRequest.properties.effective_from.format = 'date'
	$authorizationRequest.properties.effective_to.format = 'date'
	$authorizationRequest.properties.idempotency_key.minLength = 1
	$authorizationRequest.properties.source = [ordered]@{ '$ref' = '#/components/schemas/RegulatorySourceRequest' }
	$authorizationRequest.required = @('offering_id','location_id','status','authority_name','decision_reference','capacity_unit','shift','effective_from','source','idempotency_key')
}
foreach ($schemaName in @('get_api_institution_locations_item','post_api_institution_locations_response','patch_api_institution_locations_locationid_response','get_api_institution_education_offerings_item','post_api_institution_education_offerings_response','patch_api_institution_education_offerings_offeringid_response','get_api_institution_offering_authorizations_item','post_api_institution_offering_authorizations_response')) {
	$schema = $common.components.schemas[$schemaName]
	if (-not $schema) { continue }
	if ($schema.properties.id) { $schema.properties.id.format = 'uuid' }
	foreach ($field in @('offering_id','location_id','replaces_authorization_id')) { if ($schema.properties[$field]) { $schema.properties[$field].format = 'uuid' } }
	foreach ($field in @('effective_from','effective_to')) { if ($schema.properties[$field]) { $schema.properties[$field].format = 'date' } }
	if ($schema.properties.expected_version) { $schema.properties.expected_version.minimum = 1 }
	if ($schema.properties.capacity) { $schema.properties.capacity.minimum = 0 }
	if ($schema.properties.capacity_unit) { $schema.properties.capacity_unit.enum = @('students','study_groups') }
	if ($schema.properties.shift) { $schema.properties.shift.enum = @('day','afternoon','evening') }
	if ($schema.properties.status -and $schemaName -like '*authorizations*') { $schema.properties.status.enum = @('provisional','authorized','accredited','suspended','withdrawn','expired') }
}

# Reviewing an admission document is a terminal assessment command. "submitted"
# describes evidence before review and must not be exposed as an accepted command
# value, otherwise the immutable evidence row cannot be reviewed a second time.
$reviewAdmissionDocumentRequest = $common.components.schemas['ReviewApplicationDocumentRequest']
if ($reviewAdmissionDocumentRequest) {
	$reviewAdmissionDocumentRequest.properties.status.enum = @('accepted','rejected','waived')
	$reviewAdmissionDocumentRequest.properties.expected_version.minimum = 1
}

foreach ($portfolioDecisionRequestName in @('PortfolioReturnForCorrectionsRequest','PortfolioManagerialDecisionRequest')) {
	$portfolioDecisionRequest = $common.components.schemas[$portfolioDecisionRequestName]
	if (-not $portfolioDecisionRequest) { continue }
	$portfolioDecisionRequest.properties.reviewed_on.format = 'date'
	$portfolioDecisionRequest.properties.missing_documents.minimum = 0
	$portfolioDecisionRequest.properties.compliance_score.minimum = 0
	$portfolioDecisionRequest.properties.compliance_score.maximum = 100
	if ($portfolioDecisionRequestName -eq 'PortfolioManagerialDecisionRequest') {
		$portfolioDecisionRequest.properties.outcome.enum = @('acceptat','respins')
	}
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

$retentionPath = '/api/earchiva/retention-rules'
$sourcePath = '/api/regulatory-sources'
$custodyRecoveryPath = '/api/earchiva/admin/portfolio-custody-intents'
$custodyRecoveryRequest = $common.components.schemas['ReconcilePortfolioCustodyRequest']
if ($custodyRecoveryRequest) {
    $custodyRecoveryRequest.properties.disposition.enum = @('teacher_access','institution_archive_only')
    $custodyRecoveryRequest.properties.reason.minLength = 1
    $custodyRecoveryRequest.properties.title.minLength = 1
    $custodyRecoveryRequest.properties.original_file_name.minLength = 1
    $custodyRecoveryRequest.properties.document_date.format = 'date'
}
foreach ($schemaName in @(
    'get_api_earchiva_admin_portfolio_custody_intents_item',
    'post_api_earchiva_admin_portfolio_custody_intents_intentid_reconcile_response',
    'get_api_earchiva_admin_portfolio_custody_intents_intentid_recovery_operations_operationid_response'
)) {
    $schema = $common.components.schemas[$schemaName]
    if (-not $schema) { continue }
    foreach ($field in @('intent_id','operation_id','portfolio_id')) { if ($schema.properties[$field]) { $schema.properties[$field].format = 'uuid' } }
    foreach ($field in @('created_at','updated_at')) { $schema.properties[$field].format = 'date-time' }
    if ($schema.properties.document_date) { $schema.properties.document_date.format = 'date' }
    $schema.properties.status.enum = @('stored','queued','leased','committed','blocked','deadletter')
    if ($schema.properties.disposition) { $schema.properties.disposition.enum = @('teacher_access','institution_archive_only') }
}
foreach ($path in @("$custodyRecoveryPath/{intentID}/reconcile", "$custodyRecoveryPath/{intentID}/recovery-operations/{operationID}")) {
    foreach ($method in @('get','post')) {
        if (-not $paths[$path] -or -not $paths[$path].Contains($method)) { continue }
        foreach ($parameter in $paths[$path][$method].parameters) { if ($parameter.in -eq 'path') { $parameter.schema.format = 'uuid' } }
    }
}
foreach ($schemaName in @('get_api_regulatory_sources_item','post_api_regulatory_sources_response','post_api_regulatory_sources_sourceid_verify_response','post_api_regulatory_sources_sourceid_activate_response')) {
    $schema=$common.components.schemas[$schemaName]
    foreach ($field in @('id','latest_evidence_id','activation_evidence_id')) { $schema.properties[$field].format='uuid' }
    foreach ($field in @('applicable_from','applicable_until')) { $schema.properties[$field].format='date' }
    foreach ($field in @('created_at','updated_at','latest_evidence_retrieved_at','activated_at')) { $schema.properties[$field].format='date-time' }
    $schema.properties.expected_version.minimum=1
    $schema.properties.latest_evidence_sha256.pattern='^[a-f0-9]{64}$'
    $schema.properties.status.enum=@('draft','verified','active','superseded','withdrawn')
    $schema.properties.source_kind.enum=@('law','government_decision','ministerial_order','authorization','accreditation','founder_decision','contract','other')
}
$sourceRequest=$common.components.schemas['Request_post_api_regulatory_sources']
$sourceRequest.properties.source_kind.enum=@('law','government_decision','ministerial_order','authorization','accreditation','founder_decision','contract','other')
foreach ($field in @('applicable_from','applicable_until')) { $sourceRequest.properties[$field].format='date' }
$sourceRequest.properties.publisher_url.format='uri'
$sourceRequest.properties.publisher_url.description='HTTPS publisher URL without credentials, query or fragment. Host must be approved by server configuration; redirects are checked independently. A fetch does not approve legal applicability.'
foreach ($action in @('verify','activate')) {
    $schema=$common.components.schemas["Request_post_api_regulatory_sources_sourceid_${action}"]
    $schema.properties.expected_version.minimum=1
}
$activation=$common.components.schemas['Request_post_api_regulatory_sources_sourceid_activate']
$activation.properties.evidence_id.format='uuid'
$activation.properties.assessment.minLength=1
$paths[$sourcePath].post.responses['200']=$paths[$sourcePath].post.responses['201']
$paths["$sourcePath/{sourceID}/verify"].post.responses['503']=[ordered]@{
    description='Publisher retrieval is not configured. No source is marked verified.'
    content=[ordered]@{'application/json'=[ordered]@{schema=[ordered]@{type='object';required=@('code');additionalProperties=$false;properties=[ordered]@{code=[ordered]@{type='string'}}}}}
}
foreach ($path in @($sourcePath,"$sourcePath/{sourceID}/verify","$sourcePath/{sourceID}/activate","$sourcePath/{sourceID}/evidence/{evidenceID}")) {
    foreach ($method in @('get','post')) {
        if (-not $paths[$path].Contains($method)) { continue }
        foreach ($parameter in $paths[$path][$method].parameters) {
            if ($parameter.in -eq 'path') { $parameter.schema.format='uuid' }
        }
    }
}
foreach ($schemaName in @('get_api_earchiva_retention_rules_response','post_api_earchiva_retention_rules_response','post_api_earchiva_retention_rules_ruleid_approve_response','post_api_earchiva_retention_rules_ruleid_retire_response')) {
    $schema = $common.components.schemas[$schemaName]
    if ($schemaName.StartsWith('get_')) { $schema = $common.components.schemas['get_api_earchiva_retention_rules_item'] }
    foreach ($field in @('id','taxonomy_node_id','source_id')) { $schema.properties[$field].format='uuid' }
    foreach ($field in @('effective_from','effective_to')) { $schema.properties[$field].format='date' }
    foreach ($field in @('created_at','updated_at','proposed_at','approved_at','retired_at')) { $schema.properties[$field].format='date-time' }
    $schema.properties.source_checksum_sha256.pattern='^[a-f0-9]{64}$'
    $schema.properties.expected_version.minimum=1
    $schema.properties.status.enum=@('proposed','active','retired','revoked')
}
$retentionRequest=$common.components.schemas['Request_post_api_earchiva_retention_rules']
$retentionRequest.properties.anchor_kind.enum=@('intake_received_at')
$retentionRequest.properties.duration_model.enum=@('minimum_days')
$retentionRequest.properties.minimum_retention_days.type='integer'
$retentionRequest.properties.minimum_retention_days.minimum=1
$retentionRequest.properties.minimum_retention_days.maximum=36500
$retentionRequest.required=@($retentionRequest.required + 'minimum_retention_days' | Select-Object -Unique)
foreach ($field in @('taxonomy_node_id','source_id')) { $retentionRequest.properties[$field].format='uuid' }
foreach ($field in @('effective_from','effective_to')) { $retentionRequest.properties[$field].format='date' }
foreach ($action in @('approve','retire')) {
    $schema=$common.components.schemas["Request_post_api_earchiva_retention_rules_ruleid_${action}"]
    $schema.properties.expected_version.minimum=1
    if ($action -eq 'retire') { $schema.properties.reason.minLength=1 }
}
$paths[$retentionPath].post.responses['200']=$paths[$retentionPath].post.responses['201']

$artifactPath = '/api/admissions/legal-preparations/{preparationID}/artifacts/{artifactSlot}'
$portfolioUploadPath = '/api/education/portfolios/me/{recordID}/archive-documents'
if ($paths.Contains($portfolioUploadPath)) {
    $uploadOperation = $paths[$portfolioUploadPath].post
    $uploadOperation.parameters = @($uploadOperation.parameters) + @([ordered]@{
        name='Idempotency-Key'; in='header'; required=$true
        schema=[ordered]@{type='string';minLength=1;maxLength=200}
    })
    $uploadOperation.responses['200']=$uploadOperation.responses['201']
    foreach ($entry in @(@('429','Upload capacity exhausted'),@('502','Object storage unavailable'),@('503','Archive storage unconfigured'))) {
        $uploadOperation.responses[$entry[0]]=[ordered]@{description=$entry[1];content=[ordered]@{'application/json'=[ordered]@{schema=[ordered]@{type='object';required=@('code');properties=[ordered]@{code=[ordered]@{type='string'}}}}}}
    }
}
foreach ($method in @('get','post')) {
    $artifactSchema = $common.components.schemas["${method}_api_admissions_legal_preparations_preparationid_artifacts_artifactslot_response"]
    $artifactSchema.properties.document = Get-GoDTOObjectSchema 'backend/internal/earchiva/archive_documents.go|ArchiveDocument'
    $artifactSchema.properties.version = Get-GoDTOObjectSchema 'backend/internal/earchiva/archive_documents.go|ArchiveDocumentVersion'
    foreach ($field in @('intent_id','preparation_id')) { $artifactSchema.properties[$field].format = 'uuid' }
    $artifactSchema.properties.artifact_slot.enum = @('primary','resulting_decision')
    $artifactSchema.properties.retention_until.format = 'date-time'
    foreach ($field in @('received_at','created_at','updated_at')) { $artifactSchema.properties.document.properties[$field].format = 'date-time' }
    $artifactSchema.properties.version.properties.created_at.format = 'date-time'
    $artifactSchema.properties.version.properties.source_sha256.pattern = '^[a-f0-9]{64}$'
    $artifactSchema.properties.version.properties.source_size_bytes.minimum = 1
    $artifactSchema.properties.version.properties.source_size_bytes.maximum = 104857600
    if ($method -eq 'post') {
        foreach ($entry in @(@('429','Concurrent upload capacity exhausted'),@('502','Object persistence is uncertain; retry with the same file and idempotency key'),@('503','Archive storage unavailable'))) {
            $paths[$artifactPath].post.responses[$entry[0]] = [ordered]@{ description = $entry[1]; content = [ordered]@{ 'application/json' = [ordered]@{ schema = [ordered]@{ type='object'; required=@('code'); additionalProperties=$false; properties=[ordered]@{code=[ordered]@{type='string'};message=[ordered]@{type='string'}} } } } }
        }
    }
}

# Reviewed legal command constraints mirror the admission handlers. In
# particular, favorable resolutions cannot be prepared without replacement facts.
$decisionOutcomes = @('admitted','waitlisted','rejected','withdrawn','cancelled')
foreach ($preparationSchemaName in @('post_api_admissions_applications_applicationid_decision_preparations_response','post_api_admissions_appeals_appealid_resolution_preparations_response')) {
$legalPreparation = $common.components.schemas[$preparationSchemaName]
foreach ($field in @('retention_policy_id','retention_rule_version_id','retention_source_id')) {
    $legalPreparation.properties[$field].format = 'uuid'
}
foreach ($field in @('retention_anchor_at','required_retention_until')) {
    $legalPreparation.properties[$field].format = 'date-time'
}
$legalPreparation.properties.minimum_retention_days.minimum = 1
$legalPreparation.properties.minimum_retention_days.maximum = 36500
$legalPreparation.properties.required_retention_until.description = 'Immutable server-derived minimum storage deadline based on the approved retention policy and the preparation expiry. The browser cannot choose or shorten this deadline.'
}
$decisionPreparation = $common.components.schemas['PrepareDecisionRequest']
$decisionPreparation.properties.outcome.enum = $decisionOutcomes
$decisionPreparation.properties.decision_no.pattern = '^[A-Za-z0-9][A-Za-z0-9._/-]{0,63}$'
$decisionPreparation.properties.expected_version.minimum = 1
$decisionPreparation.properties.rationale.minLength = 1
$decisionPreparation.properties.appeal_deadline.format = 'date'
$appealPreparation = $common.components.schemas['PrepareAppealResolutionRequest']
$appealPreparation.properties.outcome.enum = @('upheld','partially_upheld','dismissed','withdrawn')
$appealPreparation.properties.resulting_outcome.enum = $decisionOutcomes
$appealPreparation.properties.expected_version.minimum = 1
$appealPreparation.properties.application_expected_version.minimum = 1
$appealPreparation.properties.rationale.minLength = 1
$appealPreparation.allOf = @([ordered]@{
    'if' = [ordered]@{ properties = [ordered]@{ outcome = [ordered]@{ enum = @('upheld','partially_upheld') } }; required = @('outcome') }
    then = [ordered]@{ required = @('resulting_decision_no','resulting_outcome'); properties = [ordered]@{ resulting_decision_no = [ordered]@{ type = 'string'; pattern = '^[A-Za-z0-9][A-Za-z0-9._/-]{0,63}$' } } }
})
$signerProposal = $common.components.schemas['ProposeAdmissionSignerAuthorizationRequest']
$signerProposal.properties.certificate_sha256.pattern = '^[0-9A-Fa-f]{64}$'
$signerProposal.properties.user_id.format = 'uuid'
$signerProposal.properties.permission_code.enum = @('education.admissions.decide','education.admissions.appeals.manage')
$signerProposal.properties.valid_until.format = 'date-time'
$signerProposal.properties.valid_until.description = 'Must be in the future and less than ten years from the server clock; approval rechecks expiry.'
$common.components.schemas['ApproveAdmissionSignerAuthorizationRequest'].properties.proposal_id.format = 'uuid'
$common.components.schemas['RevokeAdmissionSignerAuthorizationRequest'].properties.expected_version.minimum = 1
$common.components.schemas['RevokeAdmissionSignerAuthorizationRequest'].properties.reason.minLength = 1
$finalization = $common.components.schemas['FinalizeAdmissionLegalPreparationRequest']
$finalization.properties.preparation_id.format = 'uuid'
$finalization.properties.resulting_decision_archive.description = 'Required exactly when the server preparation includes a replacement decision. Must reference a distinct signed WORM artifact; each document is independently validated against its own canonical payload.'

$manifestDocumentDTO = Get-GoDTOObjectSchema 'backend/internal/education/portfolio_export_manifest.go|PortfolioExportManifestDocument'
foreach ($field in @('source_object_version_id', 'source_size_bytes', 'mime_type', 'zip_path')) {
    $common.components.schemas.PortfolioExportManifestDocument.properties[$field] = $manifestDocumentDTO.properties[$field]
    $common.components.schemas.PortfolioExportManifestDocument.required = @($common.components.schemas.PortfolioExportManifestDocument.required) + @($field)
}
$common.components.schemas['PortfolioExportManifestGeneratedFile'] = Get-GoDTOObjectSchema 'backend/internal/education/portfolio_export_manifest.go|PortfolioExportManifestGeneratedFile'
$common.components.schemas.PortfolioExportManifestGeneratedFile.properties.sha256.pattern = '^[0-9a-f]{64}$'
$common.components.schemas.PortfolioExportManifestGeneratedFile.properties.size_bytes.minimum = 1
$common.components.schemas.PortfolioExportManifest.properties['generated_files'] = [ordered]@{
    type = 'array'
    items = [ordered]@{ '$ref' = '#/components/schemas/PortfolioExportManifestGeneratedFile' }
    description = 'Exact generated ZIP payload hashes and byte lengths. Downloadable bundles include portfolio.json, opis.json and opis.txt; legacy manifest-only records may omit this field. Manifest and checksum sidecars are excluded to avoid circular hashes.'
}
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
		[ordered]@{name='Institutie'; description='Host-scoped institution classification and effective policy capabilities.'},
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
