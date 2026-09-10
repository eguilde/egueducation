[CmdletBinding()]
param(
    [string]$Router = "backend/cmd/server/main.go",
    [string]$Spec = "openapi/openapi.json"
)

$ErrorActionPreference = 'Stop'
if (-not (Test-Path $Spec)) { throw "OpenAPI contract not found: $Spec. Run generate-openapi.ps1 first." }

$specData = Get-Content -Raw $Spec | ConvertFrom-Json -AsHashtable
if ($specData.openapi -ne '3.1.1') { throw "Expected OpenAPI 3.1.1, got '$($specData.openapi)'" }
if (-not $specData.components.schemas.Problem -or -not $specData.components.schemas.AccessTokenAuthorizationClaims -or -not $specData.components.securitySchemes.oidcAuthorizationCode) { throw 'Required Problem, access-token authorization contract, or OIDC security scheme missing.' }

$accessClaims = $specData.components.schemas.AccessTokenAuthorizationClaims
foreach ($claim in @('sub', 'iss', 'aud', 'exp', 'iat', 'jti', 'sid', 'amr', 'acr', 'token_use', 'tenant_id', 'tenant_code', 'institution_id', 'roles', 'platform_roles', 'permissions', 'authz_version')) {
    if ($accessClaims.required -notcontains $claim -or -not $accessClaims.properties.Contains($claim)) {
        throw "Access-token authorization contract is missing required claim '$claim'"
    }
}

$sessionContract = $specData.components.schemas.SessionContext
foreach ($field in @('user', 'tenant_code', 'institution_id', 'institution_name', 'platform_roles', 'authz_version', 'permissions', 'modules', 'authentication', 'gdpr_capabilities')) {
    if ($sessionContract.required -notcontains $field -or -not $sessionContract.properties.Contains($field)) {
        throw "SessionContext contract is missing required field '$field'"
    }
}

$specRaw = Get-Content -Raw $Spec
foreach ($reference in [regex]::Matches($specRaw, '"#/components/(schemas|responses|parameters|securitySchemes)/([^"/]+)"')) {
    $kind = $reference.Groups[1].Value
    $name = $reference.Groups[2].Value
    if (-not $specData.components[$kind].Contains($name)) { throw "Dangling OpenAPI component reference: #/components/$kind/$name" }
}

$routerSource = Get-Content -Raw $Router
$matches = [regex]::Matches($routerSource, '\.(Get|Post|Put|Patch|Delete)\("([^"]+)"')
$expected = @{}
foreach ($match in $matches) {
    $method = $match.Groups[1].Value.ToLowerInvariant()
    $registeredPath = $match.Groups[2].Value
    $path = if ($registeredPath -in @('/health', '/readyz', '/healthz')) { $registeredPath } elseif ($registeredPath -eq '/logout') { '/logout' } else { "/api$registeredPath" }
    $expected["$method $path"] = $true
}

$missing = @()
$operationIds = @{}
foreach ($entry in $expected.Keys) {
    $parts = $entry.Split(' ', 2)
    $path = $parts[1]; $method = $parts[0]
    if (-not $specData.paths.Contains($path) -or -not $specData.paths[$path].Contains($method)) { $missing += $entry; continue }
    $operation = $specData.paths[$path][$method]
    if ([string]::IsNullOrWhiteSpace($operation.operationId)) { throw "Missing operationId: $entry" }
    if ($operation.tags -is [string] -or @($operation.tags).Count -eq 0) { throw "OpenAPI operation tags must be a non-empty array: $entry" }
    if ($operationIds.Contains($operation.operationId)) { throw "Duplicate operationId: $($operation.operationId)" }
    $operationIds[$operation.operationId] = $true
}
if ($missing.Count -gt 0) { throw ("OpenAPI coverage failure; missing {0} router operation(s): {1}" -f $missing.Count, ($missing -join ', ')) }

$schemaPrefixes = @('/api/registratura/', '/api/workflow/', '/api/earchiva/', '/api/auth/', '/api/passkeys/', '/api/eudi-wallet/')
$schemaExact = @('/api/me', '/api/profile')
$publicIdentity = @('/api/auth/methods', '/api/auth/ui-config', '/api/auth/role-catalog', '/api/auth/role-positions')
$incomplete = @()
foreach ($entry in $expected.Keys) {
    $parts = $entry.Split(' ', 2); $path = $parts[1]; $method = $parts[0]
    if (-not (($schemaPrefixes | Where-Object { $path.StartsWith($_) }) -or $path -in $schemaExact)) { continue }
    $operation = $specData.paths[$path][$method]
    $mustHaveSecurity = $path -notin $publicIdentity
    if ($operation.'x-contract-status' -ne 'detailed' -or ($mustHaveSecurity -and (-not $operation.security -or -not $operation.'x-tenant-scope' -or -not $operation.'x-required-permission'))) { $incomplete += $entry }
}
if ($incomplete.Count -gt 0) { throw ("Schema-tier failure; missing detailed security/RBAC/tenant metadata: " + ($incomplete -join ', ')) }

$publicPaths = @('/health', '/healthz', '/readyz', '/api/config', '/api/meta/app', '/api/auth/methods', '/api/auth/ui-config', '/api/auth/role-catalog', '/api/auth/role-positions', '/api/oidc/ui/login.js', '/api/oidc/ui/logout.js')
$placeholderOperations = @()
$securityFailures = @()
$educationFailures = @()
$genericScopedOperations = @()
foreach ($entry in $expected.Keys) {
    $parts = $entry.Split(' ', 2); $path = $parts[1]; $method = $parts[0]
    $operation = $specData.paths[$path][$method]
    if ($operation.'x-contract-status' -ne 'detailed') { $placeholderOperations += $entry }

    if ($path -notin $publicPaths) {
        if (-not $operation.security -or -not $operation.'x-tenant-scope' -or [string]::IsNullOrWhiteSpace($operation.'x-required-permission')) {
            $securityFailures += $entry
        }
    }

    if ($path.StartsWith('/api/education/')) {
        $clientSelectedInstitution = @($operation.parameters | Where-Object { $_.'$ref' -eq '#/components/parameters/Institution' -or $_.name -eq 'X-Institution-ID' })
        if ($operation.'x-contract-status' -ne 'detailed' -or $clientSelectedInstitution.Count -ne 0 -or $operation.'x-required-permission' -in @('authenticated', 'one of the permissions required by this route')) {
            $educationFailures += $entry
        }
    }

    if ($operation.security -and (($operation | ConvertTo-Json -Depth 100) -match '#/components/schemas/Entity')) { $genericScopedOperations += $entry }
}
if ($placeholderOperations.Count -gt 0) { throw ("Placeholder OpenAPI operations remain: " + ($placeholderOperations -join ', ')) }
if ($securityFailures.Count -gt 0) { throw ("Authenticated operations missing security/tenant/RBAC metadata: " + ($securityFailures -join ', ')) }
if ($educationFailures.Count -gt 0) { throw ("Education contract metadata failure: " + ($educationFailures -join ', ')) }
if ($genericScopedOperations.Count -gt 0) { throw ("Scoped operations may not use generic Entity: " + ($genericScopedOperations -join ', ')) }

$unknownSchemas = @($specData.components.schemas.GetEnumerator() | Where-Object { $_.Value.'x-schema-status' -eq 'unknown' })
if ($unknownSchemas.Count -gt 0) { throw ("Unknown OpenAPI schemas remain: " + (($unknownSchemas | ForEach-Object Key) -join ', ')) }

$genericRequestSchemas = @('IdentityRequest', 'AdminCommand', 'GdprCommand', 'Mutation', 'RegistraturaRequest')
$openRequestBodies = @()
foreach ($pathEntry in $specData.paths.GetEnumerator()) {
    foreach ($methodEntry in $pathEntry.Value.GetEnumerator()) {
        $body = $methodEntry.Value.requestBody
        if (-not $body) { continue }
        foreach ($content in $body.content.GetEnumerator()) {
            $schema = $content.Value.schema
            $reference = [string]$schema.'$ref'
            if ($reference -match '/(' + ($genericRequestSchemas -join '|') + ')$') { throw "Routed operation still references generic request schema: $($methodEntry.Key.ToUpperInvariant()) $($pathEntry.Key) -> $reference" }
            if ($schema.additionalProperties -eq $true -and -not $schema.'x-free-form-property') { $openRequestBodies += "$($methodEntry.Key.ToUpperInvariant()) $($pathEntry.Key)" }
            if ($reference) {
                $schemaName = $reference.Split('/')[-1]
                $resolved = $specData.components.schemas[$schemaName]
                if ($resolved.additionalProperties -eq $true -and -not $resolved.'x-free-form-property') { $openRequestBodies += "$($methodEntry.Key.ToUpperInvariant()) $($pathEntry.Key) -> $schemaName" }
                if ($body.required -eq $true -and $resolved.type -eq 'object' -and @($resolved.properties.Keys).Count -eq 0) { $openRequestBodies += "$($methodEntry.Key.ToUpperInvariant()) $($pathEntry.Key) -> required empty object $schemaName" }
            } elseif ($body.required -eq $true -and $schema.type -eq 'object' -and @($schema.properties.Keys).Count -eq 0) {
                $openRequestBodies += "$($methodEntry.Key.ToUpperInvariant()) $($pathEntry.Key) -> required empty inline object"
            }
        }
    }
}
if ($openRequestBodies.Count -gt 0) { throw ('Unrestricted request-body schemas remain: ' + ($openRequestBodies -join ', ')) }
if ($specData.paths['/api/eudi-wallet/activate'].post.requestBody) { throw 'EUDI wallet activation is a bodyless command; requestBody must be absent.' }
& node scripts/openapi/assert-success-response-closed.js
if ($LASTEXITCODE -ne 0) { throw 'Success response schema contains unrestricted object through a reference chain.' }

# Family response fallbacks erase meaningful SDK contracts. They must never be
# routed again: every operation must select a DTO/closed shape that mirrors its
# handler payload.
$forbiddenResponseFallbacks=@('IdentityResponse','RegistraturaResponse','WorkflowResponse','ArchiveResponse','AdminListResponse','GdprListResponse','Page')
foreach($pathEntry in $specData.paths.GetEnumerator()) { foreach($methodEntry in $pathEntry.Value.GetEnumerator()) {
    $serialized=$methodEntry.Value | ConvertTo-Json -Depth 100
    foreach($fallback in $forbiddenResponseFallbacks) { if($serialized -match "#/components/schemas/$fallback") { throw "Routed operation uses forbidden response fallback ${fallback}: $($methodEntry.Key.ToUpperInvariant()) $($pathEntry.Key)" } }
} }
foreach($fallback in $forbiddenResponseFallbacks) { if($specData.components.schemas.Contains($fallback)) { throw "Forbidden response fallback component remains: $fallback" } }

function Get-SuccessSchema([string]$path,[string]$method='get') {
    $operation=$specData.paths[$path][$method]
    $success=$operation.responses.GetEnumerator() | Where-Object { $_.Key -match '^2' } | Select-Object -First 1
    $schema=($success.Value.content.GetEnumerator() | Select-Object -First 1).Value.schema
    $reference=[string]$schema.'$ref'; if($reference){return $specData.components.schemas[$reference.Split('/')[-1]]}; return $schema
}
function Assert-SuccessProperties([string]$path,[string[]]$fields,[string]$method='get') {
    $schema=Get-SuccessSchema $path $method
    foreach($field in $fields){if(-not $schema.properties.Contains($field)){throw "Semantic response contract missing '$field': $($method.ToUpperInvariant()) $path"}}
}
function Assert-SuccessArrayItemProperties([string]$path,[string[]]$fields,[string]$method='get') {
    $schema=Get-SuccessSchema $path $method; $item=$schema.items; $reference=[string]$item.'$ref'; if($reference){$item=$specData.components.schemas[$reference.Split('/')[-1]]}
    foreach($field in $fields){if(-not $item.properties.Contains($field)){throw "Semantic array response contract missing '$field': $($method.ToUpperInvariant()) $path"}}
}
# Representative high-risk responses guard authentication/profile, registratura,
# workflow and archive semantics in addition to structural OpenAPI validity.
Assert-SuccessProperties '/api/me' @('user','tenant_code','institution_id','platform_roles','authz_version','permissions','modules','authentication')
Assert-SuccessProperties '/api/profile' @('user','tenant_code','institution_id','platform_roles','authz_version','permissions','modules','authentication') 'put'
Assert-SuccessProperties '/api/passkeys/login-options' @('status','options') 'post'
Assert-SuccessProperties '/api/passkeys/register-options' @('challenge','rp','user','pubKeyCredParams') 'post'
Assert-SuccessArrayItemProperties '/api/passkeys' @('id','credential_id','device_name','created_at','last_used_at')
Assert-SuccessProperties '/api/registratura/registre/{id}' @('id','nume','prefix_nr','nr_curent','nr_urmator','tip_registru','isDefault')
Assert-SuccessProperties '/api/registratura/parties/{id}' @('id','party_type','display_name','email','active','birth_date','legal_representative')
Assert-SuccessProperties '/api/registratura/admin/users/{id}/assignments' @('user_id','department_ids','primary_department_id','organization_id')
Assert-SuccessProperties '/api/registratura/documents/{documentID}' @('id','registru_id','registry_number','subject','direction','status','department_ids','workflow_version')
Assert-SuccessProperties '/api/registratura/documents/{documentID}/workflow-actions' @('id','status','workflow_version','workflow_assignment') 'post'
Assert-SuccessProperties '/api/workflow/dashboard' @('stats')
Assert-SuccessProperties '/api/workflow/tasks' @('id','definition_code','title','status','priority','available_actions') 'post'
Assert-SuccessProperties '/api/earchiva/records/filters' @('fonds','series','statuses','source_modules','archivists')
Assert-SuccessProperties '/api/earchiva/documents/{documentID}' @('id','title','mime_type','status','current_version_no','latest_version')

$accessTokenClaims = $specData.components.schemas.AccessTokenAuthorizationClaims
foreach ($requiredAuthorizationClaim in @('tenant_code','institution_id','roles','platform_roles','permissions','authz_version','token_use')) {
    if (-not $accessTokenClaims.properties.Contains($requiredAuthorizationClaim)) { throw "Access-token contract missing authorization claim '$requiredAuthorizationClaim'." }
}
foreach ($forbiddenIdentityClaim in @('email','email_verified','phone_number','phone_number_verified','locale','name')) {
    if ($accessTokenClaims.properties.Contains($forbiddenIdentityClaim)) { throw "Access-token contract leaks scope-controlled identity claim '$forbiddenIdentityClaim'." }
}

# The Registratura main grid is the first contract-first slice. These checks
# deliberately reject the old camelCase/UUID request body and the old implicit
# query-less/200 documentation.
$documentList = $specData.paths['/api/registratura/documents'].get
$documentListParameterNames = @($documentList.parameters | ForEach-Object { if ($_.'$ref') { ($_.'$ref' -split '/')[-1] } else { $_.name } })
$expectedDocumentListParameterNames = @('page', 'pageSize', 'limit', 'sort', 'sortBy', 'direction', 'sortDir', 'q', 'filter.registru_id', 'filter.registry_number', 'filter.external_number', 'filter.subject', 'filter.document_type', 'filter.direction', 'filter.status', 'filter.correspondent', 'filter.assigned_to', 'filter.confidentiality', 'filter.registered_at', 'filter.registered_at_from', 'filter.registered_at_to', 'filter.entry_at_from', 'filter.entry_at_to', 'filter.exit_at_from', 'filter.exit_at_to', 'filter.due_date', 'filter.due_date_from', 'filter.due_date_to')
if (Compare-Object $documentListParameterNames $expectedDocumentListParameterNames) { throw 'Registratura list query contract drifted from the handler-backed paging, sorting and filter set.' }
$documentLimit = @($documentList.parameters | Where-Object { $_.name -eq 'limit' })[0]
$documentSortBy = @($documentList.parameters | Where-Object { $_.name -eq 'sortBy' })[0]
$documentRegistryFilter = @($documentList.parameters | Where-Object { $_.name -eq 'filter.registru_id' })[0]
if ($documentLimit.schema.type -ne 'integer' -or $documentLimit.schema.maximum -ne 100 -or $documentSortBy.schema.enum -notcontains 'registry_number' -or $documentRegistryFilter.schema.format -ne 'int64') { throw 'Registratura list parameter types or sort fields do not match the handler.' }

$documentCreate = $specData.paths['/api/registratura/documents'].post
if (-not $documentCreate.responses['201'] -or $documentCreate.responses['200']) { throw 'Registratura create must document its actual 201 Created response, not 200.' }
$documentCreateRef = [string]$documentCreate.requestBody.content.'application/json'.schema.'$ref'
$documentCreateSchema = $specData.components.schemas[$documentCreateRef.Split('/')[-1]]
$documentCreateFields = @($documentCreateSchema.properties.Keys | Sort-Object)
$expectedDocumentCreateFields = @('registru_id', 'subject', 'document_type', 'direction', 'status', 'correspondent', 'assigned_to', 'correspondent_party_id', 'assigned_party_id', 'confidentiality', 'summary', 'due_date', 'external_number', 'external_number_date', 'entry_at', 'exit_at', 'activity', 'record_kind', 'department_ids') | Sort-Object
$documentRegistryIDTypes = @($documentCreateSchema.properties.registru_id.type)
if ((Compare-Object $documentCreateFields $expectedDocumentCreateFields) -or ($documentRegistryIDTypes -notcontains 'integer') -or $documentCreateSchema.properties.registru_id.format -ne 'int64' -or $documentCreateSchema.properties.Contains('registryId')) { throw 'Registratura create schema must exactly match CreateDocumentRequest snake_case fields, including int64 registru_id.' }

foreach($sample in @(
    @{path='/api/admin/users'; fields=@('id','name','email','phone','locale','status','email_verified','phone_verified','preferred_otp_channel')},
    @{path='/api/admin/roles'; fields=@('code','label')},
    @{path='/api/registratura/documents/{documentID}'; fields=@('registru_id','subject','document_type','direction','status','correspondent','assigned_to','correspondent_party_id','assigned_party_id','confidentiality','summary','due_date','change_notes','external_number','external_number_date','entry_at','exit_at','activity','record_kind','department_ids','expected_workflow_version')}
)) {
    $operation = if($specData.paths[$sample.path].patch){$specData.paths[$sample.path].patch}else{$specData.paths[$sample.path].post}
    $reference=[string](($operation.requestBody.content.GetEnumerator()|Select-Object -First 1).Value.schema.'$ref'); $schema=$specData.components.schemas[$reference.Split('/')[-1]]; $actual=@($schema.properties.Keys|Sort-Object); $sampleExpected=@($sample.fields|Sort-Object)
    if((Compare-Object $actual $sampleExpected)){throw "DTO schema property drift for $($sample.path): expected exact handler DTO fields"}
}
foreach($pathEntry in $specData.paths.GetEnumerator()){foreach($methodEntry in $pathEntry.Value.GetEnumerator()){foreach($response in $methodEntry.Value.responses.Values){foreach($content in $response.content.Values){$ref=[string]$content.schema.'$ref'; if($ref){$schema=$specData.components.schemas[$ref.Split('/')[-1]]; if($schema.properties.items.items.type -eq 'object' -and $schema.properties.items.items.additionalProperties -eq $true){throw "Anonymous unrestricted success-page item: $($methodEntry.Key.ToUpperInvariant()) $($pathEntry.Key)"}}}}}}

$logout = $specData.paths['/api/oidc/session/logout'].post
if (-not $logout -or $logout.requestBody -or $logout.security.Count -ne 1 -or -not $logout.security[0].Contains('refreshCookie') -or $logout.responses['200'].content.'application/json'.schema.'$ref' -ne '#/components/schemas/LogoutResponse') {
    throw 'OIDC browser logout contract must be refresh-cookie secured, bodyless and return LogoutResponse.'
}

$revoke = $specData.paths['/api/oidc/revoke'].post
if (-not $revoke -or $revoke.security.Count -ne 0 -or $revoke.description -notmatch 'actual token') {
    throw 'OIDC RFC 7009 revocation contract must be public-client authenticated and prohibit cookie substitution.'
}
$rpLogout = $specData.paths['/api/oidc/session/end']
if (-not $rpLogout.get -or -not $rpLogout.post -or $rpLogout.get.parameters.name -notcontains 'post_logout_redirect_uri') {
    throw 'OIDC RP-initiated logout contract must expose GET and POST exact-redirect variants.'
}

foreach ($assetPath in @('/api/oidc/ui/login.js', '/api/oidc/ui/logout.js')) {
    $asset = $specData.paths[$assetPath].get
    if (-not $asset -or $asset.security.Count -ne 0 -or $asset.requestBody -or -not $asset.responses['200'].content.'text/javascript') { throw "OIDC interaction asset contract invalid: $assetPath" }
}

$educationCoverageFragments = @(Get-ChildItem 'openapi/domains/education*.coverage.json' | Sort-Object Name | ForEach-Object { Get-Content -Raw $_.FullName | ConvertFrom-Json -AsHashtable })
$educationOperations = @($educationCoverageFragments | ForEach-Object { @($_.operations) })
$declaredEducationCount = ($educationCoverageFragments | ForEach-Object { [int]$_.scope.operationCount } | Measure-Object -Sum).Sum
$unresolvedEducationRequests = @($educationOperations | Where-Object { $_.requestBody -and -not $specData.components.schemas.Contains($_.requestBody.schema) })
$invalidEducationFragments = @($educationCoverageFragments | Where-Object { $_.Contains('validation') -and (@($_.validation.missingHandlerSources).Count -ne 0 -or @($_.validation.unknownResponseSchemas).Count -ne 0) })
if ($declaredEducationCount -ne 397 -or $educationOperations.Count -ne 397 -or $invalidEducationFragments.Count -ne 0 -or $unresolvedEducationRequests.Count -ne 0) {
	throw 'Education domain coverage drift: expected 397 handler-backed operations across all School coverage fragments with every request and response schema resolved.'
}
$coveredEducation = @{}
foreach ($operation in $educationOperations) {
    if ($coveredEducation.ContainsKey([string]$operation.operationKey)) { throw "Duplicate Education coverage operation: $($operation.operationKey)" }
    $coveredEducation[[string]$operation.operationKey] = $true
}
$routedEducation = @($expected.Keys | Where-Object { $_ -match '^\w+ /api/education/' } | ForEach-Object { $_.ToUpperInvariant().Substring(0, $_.IndexOf(' ')) + $_.Substring($_.IndexOf(' ')) })
$missingEducationCoverage = @($routedEducation | Where-Object { -not $coveredEducation.ContainsKey($_) })
$orphanedEducationCoverage = @($coveredEducation.Keys | Where-Object { -not ($routedEducation -contains $_) })
if ($missingEducationCoverage.Count -gt 0 -or $orphanedEducationCoverage.Count -gt 0) {
    throw "Education coverage/router parity failed. Missing: $($missingEducationCoverage -join ', '); orphaned: $($orphanedEducationCoverage -join ', ')"
}

$problem = $specData.components.schemas.Problem
if ($problem.required -notcontains 'code' -or @($problem.required).Count -ne 1 -or $problem.additionalProperties -ne $false) {
    throw 'Backend error contract must be the closed, machine-readable {code} envelope.'
}
foreach ($responseName in @('BadRequest','Unauthorized','Forbidden','NotFound','Validation','ServerError')) {
    $response = $specData.components.responses[$responseName]
    if (-not $response.content.'application/json' -or $response.content.'application/problem+json') {
        throw "Error response $responseName must mirror backend application/json payloads."
    }
}

$taxonomy = $specData.paths['/api/education/taxonomies'].get.responses['200'].content.'application/json'.schema.'$ref'
if ($taxonomy -ne '#/components/schemas/TaxonomyCatalogResponse' -or $specData.components.schemas.TaxonomyCatalogResponse.required -notcontains 'items') {
    throw 'Education taxonomy must return the typed {items: domain -> TaxonomyItem[]} catalogue envelope.'
}

foreach ($catalogue in @(
    @{ path='/api/education/requirements'; schema='EducationPageOfEducationRequirement'; filters=@('filter.domain','filter.priority','filter.implementation_status','filter.requirement_type') },
    @{ path='/api/education/portfolios/sections'; schema='EducationPageOfPortfolioSection'; filters=@('filter.section_code','filter.component_code','filter.label') }
)) {
    $operation = $specData.paths[$catalogue.path].get
    $responseRef = [string]$operation.responses['200'].content.'application/json'.schema.'$ref'
    if ($responseRef -ne "#/components/schemas/$($catalogue.schema)") { throw "Paged catalogue must expose the typed page response: $($catalogue.path)" }
    $pageSchema = $specData.components.schemas[$catalogue.schema]
    if ($pageSchema.required -notcontains 'items' -or $pageSchema.required -notcontains 'page' -or $pageSchema.required -notcontains 'pageSize' -or $pageSchema.required -notcontains 'total') { throw "Paged catalogue is missing its httpx.WritePage envelope contract: $($catalogue.schema)" }
    $documentedFilters = @($operation.parameters | ForEach-Object { $_.name })
    foreach ($filter in $catalogue.filters) { if ($documentedFilters -notcontains $filter) { throw "Paged catalogue omits handler filter ${filter}: $($catalogue.path)" } }
}

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
    'CreatePortfolioDocumentRequest' = @('section_code','component_code','document_title','source_scope','evidence_type','issued_on','added_on','authenticity_status')
    'CreatePortfolioChecklistItemRequest' = @('requirement_code','requirement_label','section_code','source_scope','status','last_checked_on')
    'CreatePortfolioOpisEntryRequest' = @('section_code','component_code','entry_title','source_scope','document_reference','checked_on')
    'CreatePortfolioCustodyEventRequest' = @('event_type','holder_name','holder_role','location_label','access_reason','started_on','access_mode')
    'CreatePortfolioReviewEventRequest' = @('review_stage','outcome','reviewer_name','reviewed_on')
}
foreach ($schemaName in $educationRequestRequiredFields.Keys) {
    $schema = $specData.components.schemas[$schemaName]
    $expectedRequired = @($educationRequestRequiredFields[$schemaName])
    if (-not $schema -or @($schema.required).Count -ne $expectedRequired.Count -or @($expectedRequired | Where-Object { $schema.required -notcontains $_ }).Count -gt 0) { throw "Education request required-field contract drift: $schemaName" }
    if ($schema.PSObject.Properties.Name -contains 'x-requiredness') { throw "Education request must not retain provisional requiredness: $schemaName" }
}

foreach ($operation in $educationOperations | Where-Object { $_.operationKey -like 'POST *' }) {
    $sourcePath = ([string]$operation.source) -replace ':\d+$', ''
    if (-not (Test-Path $sourcePath)) { continue }
    $source = Get-Content -Raw $sourcePath
    $handler = [regex]::Escape([string]$operation.handler)
    $body = [regex]::Match($source, "(?ms)^func\s+\(s\s+\*Service\)\s+$handler\s*\(.*?(?=^func\s|\z)")
    if (-not $body.Success -or $body.Value -notmatch 'http\.StatusCreated') { continue }
    $parts = ([string]$operation.operationKey).Split(' ', 2)
    if (-not $specData.paths[$parts[1]][$parts[0].ToLowerInvariant()].responses['201']) {
        throw "Handler-created resource is not documented as 201: $($operation.operationKey)"
    }
}

$valorifications = $specData.paths['/api/education/portfolios/records/{recordID}/valorifications']
$valorificationItem = $specData.paths['/api/education/portfolios/records/{recordID}/valorifications/{itemID}']
if (-not $valorifications.post.responses['201'] -or $valorifications.post.'x-required-permission' -notmatch 'education\.portfolios\.(school\.)?manage' -or -not $valorificationItem.patch -or -not $valorificationItem.delete.responses['204']) {
    throw 'Legacy portfolio valorification CRUD must be fully routed, typed, RBAC-protected and use 201/204 semantics.'
}

foreach ($field in @('chairperson_user_id','secretary_user_id')) {
    if ($specData.components.schemas.CreateGovernanceMeetingRequest.properties[$field].format -ne 'uuid' -or $specData.components.schemas.GovernanceMeeting.properties[$field].format -ne 'uuid') {
        throw "Governance meeting identity field is missing or not UUID: $field"
    }
    if ($specData.components.schemas.CreateGovernanceMeetingRequest.required -notcontains $field -or $specData.components.schemas.GovernanceMeeting.required -contains $field) {
        throw "Governance meeting $field must be mandatory on writes and optional on responses when omitted by encoding/json."
    }
}
foreach ($schemaName in @('CreateGovernanceMembershipRequest','GovernanceMembership')) {
    if ($specData.components.schemas[$schemaName].properties.app_user_id.format -ne 'uuid' -or $specData.components.schemas[$schemaName].required -notcontains 'app_user_id') { throw "$schemaName must expose required app_user_id as UUID." }
}
$personnelCreate = $specData.paths['/api/education/personnel/records'].post
$personnelCreateRef = [string]$personnelCreate.requestBody.content.'application/json'.schema.'$ref'
$personnelCreateSchema = $specData.components.schemas[$personnelCreateRef.Split('/')[-1]]
if (-not $personnelCreateSchema.properties.Contains('app_user_id') -or $personnelCreateSchema.properties.app_user_id.format -ne 'uuid') {
	throw 'School personnel create/update contract must expose the canonical app_user_id association as a UUID.'
}
foreach ($requiredField in @('full_name','role_title','employment_type','status','evaluation_status','mobility_stage','school_year')) {
    if ($personnelCreateSchema.required -notcontains $requiredField) { throw "School personnel request is missing handler-required field $requiredField." }
}
if ($personnelCreateSchema.required -contains 'app_user_id' -or $specData.components.schemas.PersonnelRecord.required -contains 'app_user_id') {
    throw 'Personnel app_user_id is a documented optional association and must remain optional in request and response contracts.'
}
Assert-SuccessProperties '/api/education/personnel/records' @('id','app_user_id','employee_code','full_name','institution_id') 'post'

foreach ($resource in @(
    @{ path='/api/education/classes'; item='/api/education/classes/{classID}'; schema='SchoolClass'; request='CreateSchoolClassRequest' },
    @{ path='/api/education/students'; item='/api/education/students/{studentID}'; schema='SchoolStudent'; request='CreateSchoolStudentRequest' }
)) {
    $collection = $specData.paths[$resource.path]
    $item = $specData.paths[$resource.item]
    if (-not $collection.get.responses['200'] -or -not $collection.post.responses['201'] -or -not $item.get.responses['200'] -or -not $item.patch.responses['200'] -or -not $item.delete) {
        throw "School CRUD surface is incomplete: $($resource.path)"
    }
    $requestRef = [string]$collection.post.requestBody.content.'application/json'.schema.'$ref'
    $responseRef = [string]$collection.post.responses['201'].content.'application/json'.schema.'$ref'
    if ($requestRef -ne "#/components/schemas/$($resource.request)" -or $responseRef -ne "#/components/schemas/$($resource.schema)") {
        throw "School CRUD surface is not contract-typed: $($resource.path)"
    }
}
foreach ($resource in @(
    @{ path='/api/education/class-enrolments'; item='/api/education/class-enrolments/{enrolmentID}'; schema='SchoolEnrolment' },
    @{ path='/api/education/homeroom-assignments'; item='/api/education/homeroom-assignments/{assignmentID}'; schema='SchoolHomeroomAssignment' }
)) {
    if (-not $specData.paths[$resource.path].get.responses['200'] -or -not $specData.paths[$resource.path].post.responses['201'] -or -not $specData.paths[$resource.item].get.responses['200'] -or -not $specData.paths[$resource.item].patch.responses['200'] -or -not $specData.paths[$resource.item].delete.responses['200']) {
        throw "School assignment surface is incomplete: $($resource.path)"
    }
}
$reportCatalogRef = [string]$specData.paths['/api/education/reports'].get.responses['200'].content.'application/json'.schema.'$ref'
$reportRef = [string]$specData.paths['/api/education/reports/{reportCode}'].get.responses['200'].content.'application/json'.schema.'$ref'
if ($reportCatalogRef -ne '#/components/schemas/SchoolReportCatalogResponse' -or $reportRef -ne '#/components/schemas/EducationPageOfSchoolReportRow' -or -not $specData.paths['/api/education/reports/{reportCode}/csv'].get.responses['200'].content.'text/csv' -or -not $specData.paths['/api/education/reports/{reportCode}/pdf'].get.responses['200'].content.'application/pdf') {
    throw 'Server-owned School report catalogue/JSON/CSV/PDF contracts are incomplete or untyped.'
}
$reportJSONPermission = [string]$specData.paths['/api/education/reports/{reportCode}'].get.'x-required-permission'
foreach ($format in @('csv','pdf')) {
    $exportPermission = [string]$specData.paths["/api/education/reports/{reportCode}/$format"].get.'x-required-permission'
    if ($exportPermission -notmatch 'education\.reports\.export_sensitive' -or $exportPermission -notmatch 'selected report read permission') {
        throw "School $format export must require both sensitive-export and report-specific read permission."
    }
}
if ($reportJSONPermission -match 'education\.reports\.export_sensitive') {
    throw 'School JSON reports must remain available under report-specific read permission without sensitive-export permission.'
}
if (-not $specData.paths['/api/education/signatures'].get.responses['200'] -or -not $specData.paths['/api/education/signatures'].post.responses['201'] -or -not $specData.paths['/api/education/signatures/{evidenceID}'].get.responses['200'] -or -not $specData.paths['/api/education/signatures/{evidenceID}/revalidate'].post.responses['201']) {
    throw 'Signed artifact evidence list/detail/submit/revalidate contracts are incomplete.'
}
foreach ($cockpit in @(
    @{ path='/api/education/secretariat/cockpit'; permission='education.cockpit.secretariat.read'; schema='SecretariatCockpitResponse' },
    @{ path='/api/education/hr/cockpit'; permission='education.cockpit.hr.read'; schema='HRCockpitResponse' },
    @{ path='/api/education/committee/cockpit'; permission='education.cockpit.committee.read'; schema='CommitteeCockpitResponse' },
    @{ path='/api/education/inspector/cockpit'; permission='education.cockpit.inspector.read'; schema='InspectorCockpitResponse' }
)) {
    $operation = $specData.paths[$cockpit.path].get
    if (-not $operation.responses['200'] -or $operation.'x-required-permission' -ne $cockpit.permission -or [string]$operation.responses['200'].content.'application/json'.schema.'$ref' -ne "#/components/schemas/$($cockpit.schema)") {
        throw "Operational cockpit is not fully routed, typed and permission-protected: $($cockpit.path)"
    }
    $schema = $specData.components.schemas[$cockpit.schema]
    if (-not $schema -or $schema.additionalProperties -ne $false -or $schema.required -notcontains 'institution_id') {
        throw "Operational cockpit response must be closed and institution-scoped: $($cockpit.schema)"
    }
}
$assignmentOptions = $specData.paths['/api/education/classes/assignment-options'].get
$assignmentOptionParameters = @($assignmentOptions.parameters)
$assignmentKind = $assignmentOptionParameters | Where-Object { $_.name -eq 'kind' }
if (-not $assignmentOptions.responses['200'] -or $assignmentOptions.'x-required-permission' -ne 'education.classes.manage' -or -not $assignmentKind.required -or @($assignmentKind.schema.enum).Count -ne 3 -or -not ($assignmentOptionParameters | Where-Object { $_.name -eq 'q' })) {
    throw 'School assignment options must be a typed, searchable, paged manage-only endpoint with a required kind discriminator.'
}
$assignmentOptionSchema = $specData.components.schemas.SchoolAssignmentOption
foreach ($field in @('kind','class_id','student_id','personnel_id','app_user_id','code','name')) {
    if (-not $assignmentOptionSchema.properties.Contains($field)) { throw "School assignment option contract is missing $field." }
}
$eligibleArtifacts = $specData.paths['/api/education/signatures/eligible-artifacts'].get
$eligibleVersions = $specData.paths['/api/education/signatures/eligible-archive-versions'].get
$artifactTypeParameter = @($eligibleArtifacts.parameters) | Where-Object { $_.name -eq 'artifactType' }
if ($eligibleArtifacts.'x-required-permission' -ne 'education.signatures.manage' -or $eligibleVersions.'x-required-permission' -ne 'education.signatures.manage' -or -not $artifactTypeParameter.required -or @($artifactTypeParameter.schema.enum).Count -ne 6) {
    throw 'Signature evidence selectors must be manage-only and expose the exact artifact type discriminator.'
}
$signatureSubmitRef = [string]$specData.paths['/api/education/signatures'].post.requestBody.content.'application/json'.schema.'$ref'
$signatureSubmit = $specData.components.schemas[$signatureSubmitRef.Split('/')[-1]]
foreach ($requiredField in @('storage_document_id','storage_version_id')) {
    if ($signatureSubmit.required -notcontains $requiredField) { throw "Signed evidence submit must require $requiredField." }
}
foreach ($forbiddenField in @('document_sha256','storage_bucket','storage_object_key')) {
    if ($signatureSubmit.properties.Contains($forbiddenField)) { throw "Signed evidence submit must not accept browser-owned provenance field $forbiddenField." }
}

$valorificationPurposes = @('licentiere','debut','definitivat','grad_ii','grad_i','evaluare_profesionala','mobilitate','dezvoltare_profesionala','inspectie_scolara','evaluare_externa_calitate','gradatie_merit','distinctie_premiu')

$managerialDocumentRequest = $specData.components.schemas.CreateManagerialDocumentRequest
$managerialCategories = @('diagnoza','prognoza','evidenta','planificare','raport','anexa','hotarare','procedura')
$managerialStatuses = @('draft','in_review','approved','published','archived')
if ((Compare-Object @($managerialDocumentRequest.properties.document_category.enum | Sort-Object) @($managerialCategories | Sort-Object)) -or
    (Compare-Object @($managerialDocumentRequest.properties.document_status.enum | Sort-Object) @($managerialStatuses | Sort-Object)) -or
    $managerialDocumentRequest.properties.registered_on.format -ne 'date' -or
    $managerialDocumentRequest.properties.approved_on.format -ne 'date' -or
    $managerialDocumentRequest.properties.registered_on.minLength -ne 1 -or
    $managerialDocumentRequest.properties.approved_on.minLength -ne 1 -or
    $managerialDocumentRequest.allOf[0].then.required -notcontains 'approved_on') {
    throw 'Managerial document request must publish the handler enum/date contract and conditional approval-date requirement.'
}

foreach ($schemaName in @('PortfolioValorificationPackage','CreatePortfolioValorificationPackageRequest')) {
    $schema = $specData.components.schemas[$schemaName]
    $actualPurposes = @($schema.properties.purpose.enum)
    if ($schema.required -notcontains 'purpose' -or $actualPurposes.Count -ne $valorificationPurposes.Count -or @($valorificationPurposes | Where-Object { $_ -notin $actualPurposes }).Count -gt 0) {
        throw "$schemaName must require purpose and expose the complete canonical 12-purpose enum."
    }
}
$valorificationPackageList = $specData.paths['/api/education/portfolios/records/{recordID}/valorification-packages'].get
$valorificationPackageParameters = @($valorificationPackageList.parameters)
$purposeFilter = $valorificationPackageParameters | Where-Object { $_.name -eq 'filter.purpose' }
$sortParameter = $valorificationPackageParameters | Where-Object { $_.name -eq 'sort' }
if (-not $purposeFilter -or -not $sortParameter -or $sortParameter.schema.enum -notcontains 'purpose') {
    throw 'Portfolio valorification package list must expose filter.purpose and permit server-side sorting by purpose.'
}

# The authenticated-teacher portfolio surface is intentionally separate from
# institution-wide CRUD.  Keep its public response projection and every
# lifecycle/procedure/grant/export route pinned to literal, closed contracts.
# This prevents a future generated model regression from silently removing
# fields used by the OwnPortfolio workspace while the Go handler still emits
# them.
function Assert-ResponseSchemaReference([string]$path, [string]$method, [string]$schemaName) {
    $operation = $specData.paths[$path][$method]
    if (-not $operation) { throw "Portfolio operation missing: $($method.ToUpperInvariant()) $path" }
    $success = $operation.responses.GetEnumerator() | Where-Object { $_.Key -match '^2' } | Select-Object -First 1
    $reference = [string](($success.Value.content.GetEnumerator() | Select-Object -First 1).Value.schema.'$ref')
    if ($reference -ne "#/components/schemas/$schemaName") {
        throw "Portfolio response contract drift: $($method.ToUpperInvariant()) $path must return $schemaName, got $reference"
    }
}

$portfolioRecordSchema = $specData.components.schemas.PortfolioRecord
$portfolioResponseFields = @('id','portfolio_code','owner_user_id','owner_personnel_id','owner_name','owner_role','school_year','status','section_count','last_updated_on','retention_until','activity_ceased_on','retention_period_days','legal_hold_active','legal_hold_reason','withdrawn_at','withdrawal_reason','applied_procedure_id','transfer_status','authenticity_declared','consent_captured','custodian','institution_id','notes') | Sort-Object
$actualPortfolioResponseFields = @($portfolioRecordSchema.properties.Keys | Sort-Object)
if ((Compare-Object $actualPortfolioResponseFields $portfolioResponseFields) -or $portfolioRecordSchema.additionalProperties -ne $false) {
    throw 'PortfolioRecord must remain the complete closed server response projection emitted by portfolio handlers.'
}
foreach ($field in @('id','portfolio_code','owner_name','owner_role','school_year','status','section_count','last_updated_on','retention_until','retention_period_days','legal_hold_active','transfer_status','authenticity_declared','consent_captured','custodian','institution_id','notes')) {
    if ($portfolioRecordSchema.required -notcontains $field) { throw "PortfolioRecord response is missing required emitted field: $field" }
}
foreach ($field in @('owner_user_id','owner_personnel_id','activity_ceased_on','legal_hold_reason','withdrawn_at','withdrawal_reason','applied_procedure_id')) {
    if ($portfolioRecordSchema.required -contains $field) { throw "PortfolioRecord optional response field became falsely required: $field" }
}

$ownPortfolioRequest = $specData.components.schemas.OwnPortfolioRequest
$ownPortfolioFields = @('school_year','last_updated_on','notes') | Sort-Object
if ((Compare-Object @($ownPortfolioRequest.properties.Keys | Sort-Object) $ownPortfolioFields) -or (Compare-Object @($ownPortfolioRequest.required | Sort-Object) $ownPortfolioFields) -or $ownPortfolioRequest.additionalProperties -ne $false) {
    throw 'OwnPortfolioRequest must contain only owner-editable school_year, last_updated_on and notes fields.'
}
foreach ($forbiddenField in @('owner_user_id','owner_personnel_id','owner_name','owner_role','status','transfer_status','retention_until','custodian','authenticity_declared','consent_captured')) {
    if ($ownPortfolioRequest.properties.Contains($forbiddenField)) { throw "OwnPortfolioRequest exposes server-controlled field: $forbiddenField" }
}

Assert-ResponseSchemaReference '/api/education/portfolios/me' 'get' 'EducationPageOfPortfolioRecord'
foreach ($operation in @(
    @{path='/api/education/portfolios/me'; method='post'},
    @{path='/api/education/portfolios/me/{recordID}'; method='get'},
    @{path='/api/education/portfolios/me/{recordID}'; method='patch'},
    @{path='/api/education/portfolios/me/{recordID}/submit'; method='post'},
    @{path='/api/education/portfolios/records/{recordID}/verify'; method='post'},
    @{path='/api/education/portfolios/records/{recordID}/return'; method='post'},
    @{path='/api/education/portfolios/records/{recordID}/activity-cessation'; method='post'},
    @{path='/api/education/portfolios/records/{recordID}/legal-hold'; method='post'}
)) { Assert-ResponseSchemaReference $operation.path $operation.method 'PortfolioRecord' }

foreach ($operation in @(
    @{path='/api/education/portfolios/me/{recordID}/documents'; schema='EducationPageOfPortfolioDocument'},
    @{path='/api/education/portfolios/me/{recordID}/checklist'; schema='EducationPageOfPortfolioChecklistItem'},
    @{path='/api/education/portfolios/me/{recordID}/opis'; schema='EducationPageOfPortfolioOpisEntry'},
    @{path='/api/education/portfolios/me/{recordID}/reviews'; schema='EducationPageOfPortfolioReviewEvent'},
    @{path='/api/education/portfolios/me/archive-documents'; schema='EducationPageOfPortfolioArchiveAttachment'},
    @{path='/api/education/portfolios/archive-attachment-grants'; schema='EducationPageOfPortfolioArchiveAttachmentGrant'},
    @{path='/api/education/portfolios/archive-attachment-grants/eligible-documents'; schema='EducationPageOfPortfolioArchiveAttachment'},
    @{path='/api/education/portfolios/archive-attachment-grants/eligible-users'; schema='EducationPageOfEligibleGovernanceUser'},
    @{path='/api/education/portfolios/procedures'; schema='EducationPageOfPortfolioProcedure'},
    @{path='/api/education/portfolios/procedures/{procedureID}/section-rules'; schema='EducationPageOfPortfolioProcedureSectionRule'}
)) { Assert-ResponseSchemaReference $operation.path 'get' $operation.schema }

Assert-ResponseSchemaReference '/api/education/portfolios/records/{recordID}/export-manifests' 'post' 'PortfolioExportManifestResponse'
foreach ($operation in @(
    @{path='/api/education/portfolios/procedures'; method='post'; request='CreatePortfolioProcedureRequest'; response='PortfolioProcedure'},
    @{path='/api/education/portfolios/procedures/{procedureID}'; method='patch'; request='UpdatePortfolioProcedureRequest'; response='PortfolioProcedure'},
    @{path='/api/education/portfolios/procedures/{procedureID}/section-rules'; method='put'; request='ReplacePortfolioProcedureSectionRulesRequest'; response='PortfolioProcedureSectionRulesReplaceResponse'}
)) {
    $requestReference = [string]$specData.paths[$operation.path][$operation.method].requestBody.content.'application/json'.schema.'$ref'
    if ($requestReference -ne "#/components/schemas/$($operation.request)") { throw "Portfolio procedure request contract drift: $($operation.method.ToUpperInvariant()) $($operation.path)" }
    Assert-ResponseSchemaReference $operation.path $operation.method $operation.response
}

$actualCount = $expected.Count
if ($actualCount -ne 560) { throw "Router extraction drift: expected 560 concrete operations, found $actualCount. Update this guard intentionally after auditing the router." }
Write-Host "OpenAPI validation passed: $actualCount concrete router operations covered; $($operationIds.Count) unique operation IDs; detailed handler-backed contracts only; no generic Entity in scoped operations; security/tenant/RBAC metadata complete; 396 Education operations schema-complete."
