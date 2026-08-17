param(
    [string]$InventoryPath = "docs/earchiva-balotesti/balotesti-pdf-manifest.json",
    [string]$OutputPath = "docs/earchiva-balotesti/balotesti-import-manifest.json"
)

$ErrorActionPreference = 'Stop'
$inventory = Get-Content -LiteralPath $InventoryPath -Raw | ConvertFrom-Json
$inventory = @($inventory | Sort-Object @{ Expression = {
    [regex]::Replace(([string]$_.source_relative_path).ToLowerInvariant(), '\d+', { param($m) $m.Value.PadLeft(20, '0') })
} })
$files = @(); $sequence = 0; [long]$bytes = 0; [long]$pages = 0
foreach ($entry in $inventory) {
    $sequence++; $sha = ([string]$entry.sha256).ToLowerInvariant(); $size = [long]$entry.size_bytes
    $bytes += $size; $pages += [int]$entry.pages
    $files += [ordered]@{
        sequence = $sequence; source_relative_path = [string]$entry.source_relative_path
        original_filename = [string]$entry.source_relative_path; canonical_filename = ('balotesti-archive-{0:D4}.pdf' -f $sequence)
        size_bytes = $size; mime_type = 'application/pdf'; sha256 = $sha
        # No tenant/institution identifier may be supplied by an import client.
        idempotency_key = "archive-import-v1:$sha"; pages = [int]$entry.pages
        encrypted = [Convert]::ToBoolean([string]$entry.encrypted); valid = [Convert]::ToBoolean([string]$entry.valid)
        text_layer_pages = [int]$entry.text_layer_pages; text_layer_present = [Convert]::ToBoolean([string]$entry.text_layer_present)
        ingestion_state = 'discovered'
    }
}
$manifest = [ordered]@{
    manifest_version = '1.0'; batch_id = 'balotesti-legacy-scan-2026-08-17'
    # Local safety guard only; it is never sent to the API as tenant context.
    expected_api_host = 'scoalabalotesti.eguilde.cloud'
    source_inventory_sha256 = (Get-FileHash -LiteralPath $InventoryPath -Algorithm SHA256).Hash.ToLowerInvariant()
    source_policy = 'read-only; originals are never renamed, moved, recompressed, or deleted'
    canonical_naming = 'balotesti-archive-NNNN.pdf'; file_count = $files.Count; total_size_bytes = $bytes; total_pages = $pages; files = $files
}
$parent = Split-Path -Parent $OutputPath; if ($parent) { New-Item -ItemType Directory -Force -Path $parent | Out-Null }
$manifest | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $OutputPath -Encoding utf8NoBOM
$digest = (Get-FileHash -LiteralPath $OutputPath -Algorithm SHA256).Hash.ToLowerInvariant()
Set-Content -LiteralPath "$OutputPath.sha256" -Value "$digest  $(Split-Path -Leaf $OutputPath)" -Encoding ascii
[ordered]@{ manifest=$OutputPath; sha256=$digest; files=$files.Count; bytes=$bytes; pages=$pages } | ConvertTo-Json -Compress
