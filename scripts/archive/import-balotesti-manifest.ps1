<#
.SYNOPSIS
  Resumable, locally verified eArhiva importer with OIDC Authorization Code + PKCE.

.DESCRIPTION
  The importer never writes to MinIO. Each file is sent to the authenticated
  egueducation API so PDF validation, ClamAV, audit, storage and OCR/indexing
  remain server-enforced. Tenant and institution are never request parameters:
  they are derived by the API from the OIDC access token and tenant host.
#>
[CmdletBinding()]
param(
    [string]$ManifestPath = 'docs/earchiva-balotesti/balotesti-import-manifest.json',
    [string]$SourceRoot = 'D:\balotesti\scanari dosare Balotesti',
    [string]$ApiBaseUrl,
    [string]$OIDCIssuer,
    [string]$ClientID = 'egueducation-desktop',
    [string]$RedirectUri = 'http://localhost:4300/callback',
    [string]$CheckpointPath = 'docs/earchiva-balotesti/balotesti-import-checkpoint.json',
    [ValidateRange(1,999999)][int]$StartSequence = 1,
    [ValidateRange(0,999999)][int]$MaxFiles = 0,
    [switch]$Resume, [switch]$DryRun, [switch]$PollUntilTerminal,
    [ValidateRange(1,3600)][int]$PollIntervalSeconds = 15,
    [ValidateRange(1,1440)][int]$PollTimeoutMinutes = 90,
    [ValidateRange(0,10)][int]$MaxRetries = 4,
    [ValidateRange(1,300)][int]$RetryBaseDelaySeconds = 2,
    [switch]$ContinueOnError, [switch]$SelfTest
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Get-Sha256([string]$Path) { (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant() }
function Get-RandomB64Url([int]$Bytes = 32) { $b = New-Object byte[] $Bytes; [Security.Cryptography.RandomNumberGenerator]::Fill($b); [Convert]::ToBase64String($b).TrimEnd('=').Replace('+','-').Replace('/','_') }
function Get-PKCEChallenge([string]$Verifier) { $b=[Text.Encoding]::ASCII.GetBytes($Verifier); $h=[Security.Cryptography.SHA256]::HashData($b); [Convert]::ToBase64String($h).TrimEnd('=').Replace('+','-').Replace('/','_') }
function Assert-HttpsOrLoopback([string]$Raw, [string]$Label) {
    $u=[Uri]$Raw; if (-not $u.IsAbsoluteUri -or ($u.Scheme -ne 'https' -and -not ($u.Scheme -eq 'http' -and $u.Host -in @('localhost','127.0.0.1')))) { throw "$Label must use HTTPS (HTTP only for localhost tests)." }; return $u
}
function Assert-SafeRelativePath([string]$RelativePath) {
    if ([string]::IsNullOrWhiteSpace($RelativePath) -or [IO.Path]::IsPathRooted($RelativePath)) { throw 'Manifest source path is invalid.' }
    if (@($RelativePath -split '[\\/]' | Where-Object { $_ -eq '' -or $_ -eq '..' }).Count) { throw 'Manifest source path contains traversal.' }
}
function Get-SourceFile([string]$Root,[string]$RelativePath) {
    Assert-SafeRelativePath $RelativePath; $fullRoot=[IO.Path]::GetFullPath($Root)
    $candidate=[IO.Path]::GetFullPath((Join-Path $fullRoot $RelativePath)); $prefix=$fullRoot.TrimEnd([IO.Path]::DirectorySeparatorChar,[IO.Path]::AltDirectorySeparatorChar)+[IO.Path]::DirectorySeparatorChar
    if (-not $candidate.StartsWith($prefix,[StringComparison]::OrdinalIgnoreCase) -or -not (Test-Path -LiteralPath $candidate -PathType Leaf)) { throw 'Manifest source is unavailable outside or below SourceRoot.' }; $candidate
}
function Assert-ManifestEntry([object]$Entry,[string]$Root) {
    $n=[int]$Entry.sequence; if ($n -lt 1 -or -not [bool]$Entry.valid -or [bool]$Entry.encrypted -or [string]$Entry.mime_type -ne 'application/pdf') { throw "Manifest sequence $n is not eligible." }
    if ([string]$Entry.canonical_filename -ne ('balotesti-archive-{0:D4}.pdf' -f $n) -or [string]$Entry.sha256 -notmatch '^[a-fA-F0-9]{64}$') { throw "Manifest sequence $n is malformed." }
    $source=Get-SourceFile $Root ([string]$Entry.source_relative_path)
    if ((Get-Item -LiteralPath $source).Length -ne [int64]$Entry.size_bytes -or -not (Get-Sha256 $source).Equals(([string]$Entry.sha256),[StringComparison]::OrdinalIgnoreCase)) { throw "Manifest verification failed for sequence $n." }; $source
}
function Get-IdempotencyKey([object]$Entry) { 'archive-import-v1:' + ([string]$Entry.sha256).ToLowerInvariant() }
function New-Metadata([object]$Manifest,[object]$Entry,[string]$Digest) { [ordered]@{ original_filename=[string]$Entry.original_filename; canonical_filename=[string]$Entry.canonical_filename; import_provenance='balotesti-scanari-dosare'; import_batch_id=[string]$Manifest.batch_id; import_manifest_sha256=$Digest; source_sha256=([string]$Entry.sha256).ToLowerInvariant(); source_page_count=[int]$Entry.pages } }
function Get-SafeApiCode([string]$Body) { try { $code=[string](($Body|ConvertFrom-Json).code); if ($code -match '^[a-z0-9_-]{1,100}$') { return $code } } catch {}; 'http_error' }
function New-CheckpointEntry([object]$Entry,[string]$State,$Status,[string]$DocumentID,[string]$ArchiveStatus,[string]$ErrorCode) { [ordered]@{ sequence=[int]$Entry.sequence; canonical_filename=[string]$Entry.canonical_filename; sha256=([string]$Entry.sha256).ToLowerInvariant(); state=$State; http_status=$Status; document_id=$DocumentID; archive_status=$ArchiveStatus; error_code=$ErrorCode; updated_at_utc=[DateTime]::UtcNow.ToString('o') } }
function Read-Checkpoint([string]$Path) { if (-not (Test-Path -LiteralPath $Path)) { return [ordered]@{manifest_sha256='';entries=@{}} }; $c=Get-Content -LiteralPath $Path -Raw|ConvertFrom-Json; $r=@{}; foreach($e in @($c.entries)){if($null -ne $e.sequence){$r[[int]$e.sequence]=$e}}; [ordered]@{manifest_sha256=[string]$c.manifest_sha256;entries=$r} }
function Write-Checkpoint([string]$Path,[hashtable]$Entries,[string]$Digest) {
    # Projection is deliberately a closed allow-list: no response, object key, filename or token persists.
    $out=@($Entries.Values|Sort-Object sequence|ForEach-Object {[ordered]@{sequence=[int]$_.sequence;canonical_filename=[string]$_.canonical_filename;sha256=[string]$_.sha256;state=[string]$_.state;http_status=$_.http_status;document_id=[string]$_.document_id;archive_status=[string]$_.archive_status;error_code=[string]$_.error_code;updated_at_utc=[string]$_.updated_at_utc}})
    $dir=Split-Path -Parent $Path;if($dir){New-Item -ItemType Directory -Force -Path $dir|Out-Null}; $tmp="$Path.tmp"; [ordered]@{checkpoint_version='1.1';manifest_sha256=$Digest;entries=$out}|ConvertTo-Json -Depth 4|Set-Content -LiteralPath $tmp -Encoding utf8NoBOM; Move-Item -LiteralPath $tmp -Destination $Path -Force
}
function Get-JwtCnfJkt([string]$Token) {
    $parts=$Token -split '\.'; if($parts.Count -ne 3){return ''}; try{$payload=$parts[1].Replace('-','+').Replace('_','/');$payload += '=' * ((4-$payload.Length%4)%4);$json=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($payload));[string](($json|ConvertFrom-Json).cnf.jkt)}catch{''}
}
function Get-Discovery([System.Net.Http.HttpClient]$Client,[string]$Issuer) {
    $i=Assert-HttpsOrLoopback $Issuer 'OIDC issuer'; $raw=$Client.GetStringAsync($i.AbsoluteUri.TrimEnd('/')+'/.well-known/openid-configuration').GetAwaiter().GetResult(); $d=$raw|ConvertFrom-Json
    if ([string]$d.issuer -ne $i.AbsoluteUri.TrimEnd('/') -or [string]$d.authorization_endpoint -eq '' -or [string]$d.token_endpoint -eq '') { throw 'OIDC discovery is incomplete or issuer mismatches configured issuer.' }
    Assert-HttpsOrLoopback ([string]$d.authorization_endpoint) 'OIDC authorization endpoint'|Out-Null; Assert-HttpsOrLoopback ([string]$d.token_endpoint) 'OIDC token endpoint'|Out-Null
    if (@($d.response_types_supported) -notcontains 'code' -or @($d.code_challenge_methods_supported) -notcontains 'S256') { throw 'OIDC provider does not advertise Authorization Code with S256 PKCE.' }; $d
}
function Wait-LoopbackAuthorization([string]$AuthorizationUrl,[Uri]$Redirect,[string]$ExpectedState) {
    if ($Redirect.Scheme -ne 'http' -or $Redirect.Host -notin @('localhost','127.0.0.1')) { throw 'Desktop redirect must be an HTTP loopback URI.' }
    $listener=[Net.HttpListener]::new(); $prefix='http://'+$Redirect.Host+':'+$Redirect.Port+'/'; $listener.Prefixes.Add($prefix)
    try { $listener.Start() } catch { throw "Cannot bind OIDC loopback callback $prefix. The registered desktop redirect must be available." }
    try {
        Start-Process $AuthorizationUrl
        $task=$listener.GetContextAsync(); if(-not $task.Wait([TimeSpan]::FromMinutes(10))){throw 'OIDC login timed out waiting for loopback callback.'}; $ctx=$task.Result
        $received=[Uri]$ctx.Request.Url; $validPath=$received.AbsolutePath.TrimEnd('/') -eq $Redirect.AbsolutePath.TrimEnd('/'); $parameters=[System.Web.HttpUtility]::ParseQueryString($received.Query)
        $callbackState=[string]$parameters['state']; $code=[string]$parameters['code']
        if(-not $validPath -or $callbackState -ne $ExpectedState -or [string]::IsNullOrWhiteSpace($code)) { $ctx.Response.StatusCode=400;$ctx.Response.Close();throw 'OIDC callback state or path validation failed.' }
        $body=[Text.Encoding]::UTF8.GetBytes('<!doctype html><title>Autentificare reușită</title><p>Poți reveni la importator.</p>');$ctx.Response.StatusCode=200;$ctx.Response.ContentType='text/html; charset=utf-8';$ctx.Response.OutputStream.Write($body,0,$body.Length);$ctx.Response.Close();$code
    } finally { $listener.Stop();$listener.Close() }
}
function Invoke-TokenRequest([System.Net.Http.HttpClient]$Client,[string]$Endpoint,[hashtable]$Form) {
    $body=[System.Net.Http.FormUrlEncodedContent]::new($Form);$response=$Client.PostAsync($Endpoint,$body).GetAwaiter().GetResult();try{$raw=$response.Content.ReadAsStringAsync().GetAwaiter().GetResult();if(-not $response.IsSuccessStatusCode){throw ('OIDC token endpoint returned '+[int]$response.StatusCode+' ('+(Get-SafeApiCode $raw)+').')};$token=$raw|ConvertFrom-Json;if([string]::IsNullOrWhiteSpace([string]$token.access_token)){throw 'OIDC token response did not contain an access token.'};$cnf=Get-JwtCnfJkt ([string]$token.access_token);if([string]$token.token_type -ieq 'DPoP' -or $cnf){throw 'OIDC issued a DPoP-bound token. This importer fails closed because it has not negotiated a proof key.'};if([string]$token.token_type -and [string]$token.token_type -ine 'Bearer'){throw 'OIDC issued an unsupported access-token type.'};@{access_token=[string]$token.access_token;expires_at=[DateTime]::UtcNow.AddSeconds([Math]::Max(60,[int]$token.expires_in))}}finally{$response.Dispose();$body.Dispose()}
}
function Connect-OIDCDesktop([System.Net.Http.HttpClient]$Client,[string]$Issuer,[string]$ClientID,[string]$RedirectUri) {
    $d=Get-Discovery $Client $Issuer;$redirect=[Uri]$RedirectUri;$state=Get-RandomB64Url 32;$verifier=Get-RandomB64Url 64;$query=[Web.HttpUtility]::ParseQueryString('');$query['client_id']=$ClientID;$query['redirect_uri']=$redirect.AbsoluteUri;$query['response_type']='code';$query['scope']='openid profile email phone offline_access';$query['state']=$state;$query['code_challenge']=Get-PKCEChallenge $verifier;$query['code_challenge_method']='S256';$code=Wait-LoopbackAuthorization (([string]$d.authorization_endpoint)+'?'+$query.ToString()) $redirect $state
    Invoke-TokenRequest $Client ([string]$d.token_endpoint) @{grant_type='authorization_code';code=$code;redirect_uri=$redirect.AbsoluteUri;client_id=$ClientID;code_verifier=$verifier}
}
function Get-AccessToken([hashtable]$Session,[System.Net.Http.HttpClient]$Client,[string]$TokenEndpoint,[string]$ClientID) {
    if([DateTime]::UtcNow -lt $Session.expires_at.AddSeconds(-120)){return [string]$Session.access_token};$fresh=Invoke-TokenRequest $Client $TokenEndpoint @{grant_type='refresh_token';refresh_token='cookie';client_id=$ClientID};$Session.access_token=$fresh.access_token;$Session.expires_at=$fresh.expires_at;[string]$Session.access_token
}
function Invoke-Api([System.Net.Http.HttpClient]$Client,[hashtable]$Session,[string]$TokenEndpoint,[string]$ClientID,[System.Net.Http.HttpRequestMessage]$Request) {
    $Request.Headers.Authorization=[System.Net.Http.Headers.AuthenticationHeaderValue]::new('Bearer',(Get-AccessToken $Session $Client $TokenEndpoint $ClientID));$Client.SendAsync($Request).GetAwaiter().GetResult()
}
function Invoke-Upload([System.Net.Http.HttpClient]$Client,[hashtable]$Session,[string]$TokenEndpoint,[string]$ClientID,[string]$Endpoint,[string]$Source,[object]$Manifest,[object]$Entry,[string]$Digest,[int]$Retries,[int]$BaseDelay) {
    $metadata=New-Metadata $Manifest $Entry $Digest|ConvertTo-Json -Compress;for($attempt=0;$attempt -le $Retries;$attempt++){$content=$null;$response=$null;try{$content=[Net.Http.MultipartFormDataContent]::new();$stream=[IO.File]::OpenRead($Source);$file=[Net.Http.StreamContent]::new($stream);$file.Headers.ContentType=[Net.Http.Headers.MediaTypeHeaderValue]::Parse('application/pdf');$content.Add($file,'file',[string]$Entry.canonical_filename);foreach($p in @{title=('Document arhivă '+([string]$Entry.canonical_filename -replace '\.pdf$',''));source_kind='import';source_system='balotesti-manifest-importer';metadata=$metadata}.GetEnumerator()){$content.Add([Net.Http.StringContent]::new([string]$p.Value,[Text.Encoding]::UTF8),[string]$p.Key)};$request=[Net.Http.HttpRequestMessage]::new('POST',$Endpoint);$request.Headers.Add('Idempotency-Key',(Get-IdempotencyKey $Entry));$request.Content=$content;$response=Invoke-Api $Client $Session $TokenEndpoint $ClientID $request;$raw=$response.Content.ReadAsStringAsync().GetAwaiter().GetResult();$status=[int]$response.StatusCode;if($status -in 200,201){$d=$raw|ConvertFrom-Json;return @{ok=$true;status=$status;id=[string]$d.id;archive_status=[string]$d.status;code=''}};$retry=$status -eq 429 -or $status -ge 500;if(-not $retry -or $attempt -ge $Retries){return @{ok=$false;status=$status;id='';archive_status='';code=(Get-SafeApiCode $raw)}};Start-Sleep -Seconds ([Math]::Min(300,$BaseDelay*[Math]::Pow(2,$attempt)))}catch{if($attempt -ge $Retries){return @{ok=$false;status=$null;id='';archive_status='';code='network_error'}};Start-Sleep -Seconds ([Math]::Min(300,$BaseDelay*[Math]::Pow(2,$attempt)))}finally{if($response){$response.Dispose()};if($content){$content.Dispose()}}};throw 'unreachable'
}
function Wait-ArchiveTerminal([System.Net.Http.HttpClient]$Client,[hashtable]$Session,[string]$TokenEndpoint,[string]$ClientID,[string]$Endpoint,[string]$DocumentID,[int]$Interval,[int]$Timeout) {
    $until=[DateTime]::UtcNow.AddMinutes($Timeout);while([DateTime]::UtcNow -lt $until){$request=[Net.Http.HttpRequestMessage]::new('GET',$Endpoint+'/'+[Uri]::EscapeDataString($DocumentID));$response=Invoke-Api $Client $Session $TokenEndpoint $ClientID $request;try{if(-not $response.IsSuccessStatusCode){return @{terminal=$false;status='';code='poll_http_error'}};$document=$response.Content.ReadAsStringAsync().GetAwaiter().GetResult()|ConvertFrom-Json;$s=[string]$document.status;if($s -in 'ready','failed'){return @{terminal=$true;status=$s;code=''}}}finally{$response.Dispose()};Start-Sleep -Seconds $Interval};@{terminal=$false;status='';code='poll_timeout'}
}
function Invoke-SelfTest {
    $e=[pscustomobject]@{sha256=('a'*64);sequence=1;canonical_filename='balotesti-archive-0001.pdf';original_filename='original.pdf';pages=1};if((Get-IdempotencyKey $e) -match 'tenant|institution' -or (Get-PKCEChallenge 'test') -ne 'n4bQgYhMfWWaL-qgxVrQFaO_TxsrC4Is0V1sFbDwCgg'){throw 'OIDC self-test failed.'};$m=New-Metadata ([pscustomobject]@{batch_id='batch';tenant_code='never-send';institution_id='never-send'}) $e ('b'*64);if($m.Contains('tenant_code') -or $m.Contains('institution_id')){throw 'Metadata leaked tenant context.'};$c=New-CheckpointEntry $e 'accepted' 201 'doc-1' 'queued' ''|ConvertTo-Json -Compress;if($c -match 'original.pdf|token|object_key|bucket'){throw 'Checkpoint leakage self-test failed.'};@{self_test='passed';assertions=4}|ConvertTo-Json -Compress
}

if($SelfTest){Invoke-SelfTest;exit 0};if([string]::IsNullOrWhiteSpace($ApiBaseUrl)){throw 'ApiBaseUrl is required.'};$api=Assert-HttpsOrLoopback $ApiBaseUrl 'ApiBaseUrl';if(-not $OIDCIssuer){$OIDCIssuer=$api.AbsoluteUri.TrimEnd('/')+'/api/oidc'};if(-not(Test-Path -LiteralPath $ManifestPath)){throw 'ManifestPath does not exist.'};if(-not(Test-Path -LiteralPath $SourceRoot -PathType Container)){throw 'SourceRoot does not exist.'}
$manifest=Get-Content -LiteralPath $ManifestPath -Raw|ConvertFrom-Json;if([string]$manifest.manifest_version -ne '1.0'){throw 'Unsupported manifest.'};if(-not $DryRun -and [string]$manifest.expected_api_host -ne $api.Host){throw 'ApiBaseUrl host does not match the archive manifest destination guard.'};$digest=Get-Sha256 $ManifestPath;$entries=@($manifest.files|Sort-Object {[int]$_.sequence}|Where-Object {[int]$_.sequence -ge $StartSequence});if($MaxFiles){$entries=@($entries|Select-Object -First $MaxFiles)};if($DryRun){foreach($e in $entries){[void](Assert-ManifestEntry $e $SourceRoot)};@{mode='dry_run';verified=$entries.Count;uploaded=0;manifest_sha256=$digest}|ConvertTo-Json -Compress;exit 0}
$checkpointEntries=@{};if($Resume){$checkpoint=Read-Checkpoint $CheckpointPath;if($checkpoint.manifest_sha256 -and -not $checkpoint.manifest_sha256.Equals($digest,[StringComparison]::OrdinalIgnoreCase)){throw 'Checkpoint belongs to a different manifest.'};$checkpointEntries=$checkpoint.entries};$jar=[Net.CookieContainer]::new();$handler=[Net.Http.HttpClientHandler]::new();$handler.UseCookies=$true;$handler.CookieContainer=$jar;$client=[Net.Http.HttpClient]::new($handler);$client.Timeout=[TimeSpan]::FromMinutes(15)
try{$session=Connect-OIDCDesktop $client $OIDCIssuer $ClientID $RedirectUri;$discovery=Get-Discovery $client $OIDCIssuer;$endpoint=$api.AbsoluteUri.TrimEnd('/')+'/api/earchiva/documents';$summary=@{attempted=0;accepted=0;ready=0;failed=0;skipped=0};foreach($e in $entries){$old=$checkpointEntries[[int]$e.sequence];if($Resume -and $old -and [string]$old.sha256 -eq [string]$e.sha256 -and [string]$old.state -in 'accepted','ready'){$summary.skipped++;continue};$source=Assert-ManifestEntry $e $SourceRoot;$summary.attempted++;Write-Output ('Importing sequence {0}: {1}' -f $e.sequence,$e.canonical_filename);$r=Invoke-Upload $client $session ([string]$discovery.token_endpoint) $ClientID $endpoint $source $manifest $e $digest $MaxRetries $RetryBaseDelaySeconds;if(-not $r.ok){$summary.failed++;$checkpointEntries[[int]$e.sequence]=New-CheckpointEntry $e 'failed' $r.status '' '' $r.code;Write-Checkpoint $CheckpointPath $checkpointEntries $digest;if(-not $ContinueOnError){throw "Upload failed at sequence $($e.sequence): $($r.code)"};continue};$summary.accepted++;$state='accepted';if($PollUntilTerminal){$poll=Wait-ArchiveTerminal $client $session ([string]$discovery.token_endpoint) $ClientID $endpoint $r.id $PollIntervalSeconds $PollTimeoutMinutes;if(-not $poll.terminal){$checkpointEntries[[int]$e.sequence]=New-CheckpointEntry $e 'accepted' $r.status $r.id '' $poll.code;Write-Checkpoint $CheckpointPath $checkpointEntries $digest;if(-not $ContinueOnError){throw "Polling failed at sequence $($e.sequence): $($poll.code)"}}else{$state=$poll.status;if($state -eq 'ready'){$summary.ready++}else{$summary.failed++;if(-not $ContinueOnError){$checkpointEntries[[int]$e.sequence]=New-CheckpointEntry $e 'failed' $r.status $r.id 'failed' '';Write-Checkpoint $CheckpointPath $checkpointEntries $digest;throw "Archive processing failed at sequence $($e.sequence)."}}}};$checkpointEntries[[int]$e.sequence]=New-CheckpointEntry $e $state $r.status $r.id $(if($state -in 'ready','failed'){$state}else{$r.archive_status}) '';Write-Checkpoint $CheckpointPath $checkpointEntries $digest};$summary|ConvertTo-Json -Compress}finally{$client.Dispose();$handler.Dispose()}
