[CmdletBinding()]
param(
    [string]$Router = "backend/cmd/server/main.go",
    [string]$Spec = "openapi/openapi.json"
)

$ErrorActionPreference = 'Stop'
if (-not (Test-Path $Spec)) { throw "OpenAPI contract not found: $Spec. Run generate-openapi.ps1 first." }

$specData = Get-Content -Raw $Spec | ConvertFrom-Json -AsHashtable
if ($specData.openapi -ne '3.1.1') { throw "Expected OpenAPI 3.1.1, got '$($specData.openapi)'" }
if (-not $specData.components.schemas.Problem -or -not $specData.components.securitySchemes.oidcAuthorizationCode) { throw 'Required Problem schema or OIDC security scheme missing.' }

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
Assert-SuccessProperties '/api/me' @('user','institution_id','permissions','modules','authentication')
Assert-SuccessProperties '/api/profile' @('user','institution_id','permissions','modules','authentication') 'put'
Assert-SuccessProperties '/api/passkeys/login-options' @('status','options') 'post'
Assert-SuccessProperties '/api/passkeys/register-options' @('challenge','rp','user','pubKeyCredParams') 'post'
Assert-SuccessArrayItemProperties '/api/passkeys' @('id','credential_id','device_name','created_at','last_used_at')
Assert-SuccessProperties '/api/registratura/registre/{id}' @('id','nume','prefix_nr','nr_curent','nr_urmator','tip_registru','isDefault')
Assert-SuccessProperties '/api/registratura/parties/{id}' @('id','party_type','display_name','email','active','birth_date','legal_representative')
Assert-SuccessProperties '/api/registratura/admin/users/{id}/assignments' @('user_id','department_ids','primary_department_id','organization_id')
Assert-SuccessProperties '/api/registratura/documents/{documentID}' @('id','registru_id','registry_number','subject','direction','status','department_ids','workflow_version')
Assert-SuccessProperties '/api/workflow/dashboard' @('stats')
Assert-SuccessProperties '/api/workflow/tasks' @('id','definition_code','title','status','priority','available_actions') 'post'
Assert-SuccessProperties '/api/earchiva/records/filters' @('fonds','series','statuses','source_modules','archivists')
Assert-SuccessProperties '/api/earchiva/documents/{documentID}' @('id','title','mime_type','status','current_version_no','latest_version')

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

$educationCoverage = Get-Content -Raw 'openapi/domains/education.coverage.json' | ConvertFrom-Json -AsHashtable
$unresolvedEducationRequests = @($educationCoverage.operations | Where-Object { $_.requestBody -and -not $specData.components.schemas.Contains($_.requestBody.schema) })
if ($educationCoverage.operations.Count -ne 304 -or @($educationCoverage.validation.missingHandlerSources).Count -ne 0 -or $unresolvedEducationRequests.Count -ne 0 -or @($educationCoverage.validation.unknownResponseSchemas).Count -ne 0) {
    throw 'Education domain coverage drift: expected 304 handler-backed operations with all request and response schemas resolved.'
}

$actualCount = $expected.Count
if ($actualCount -ne 455) { throw "Router extraction drift: expected 455 concrete operations, found $actualCount. Update this guard intentionally after auditing the router." }
Write-Host "OpenAPI validation passed: $actualCount concrete router operations covered; $($operationIds.Count) unique operation IDs; detailed handler-backed contracts only; no generic Entity in scoped operations; security/tenant/RBAC metadata complete; 304 Education operations schema-complete."
