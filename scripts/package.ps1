# Hypo release/deploy packaging. Builds the frontend, then the Windows exe
# (so the real UI is embedded, never the tracked dist placeholder), then a
# self-contained deploy zip with the Mole engine beside the executable.
#
# Usage (Linux/macOS dev box):
#   pwsh -NoProfile -File scripts/package.ps1              # zip only
#   pwsh -NoProfile -File scripts/package.ps1 -OutDir X    # custom output
#
# Requires: node/npm, go (GOOS=windows), a Mole checkout beside this repo.
#Requires -Version 5.1
[CmdletBinding()]
param(
    [string]$OutDir = ""
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repoRoot = Split-Path -Parent $PSScriptRoot
$engineDir = Join-Path (Split-Path -Parent $repoRoot) "Mole"
if (-not (Test-Path (Join-Path $engineDir "mole.ps1"))) {
    throw "Mole engine checkout not found at $engineDir"
}

$stamp = Get-Date -Format "yyyyMMdd-HHmm"
$stage = if ($OutDir) { Join-Path $OutDir "hypo-win64" } else { Join-Path ([System.IO.Path]::GetTempPath()) "hypo-win64-$stamp" }

Write-Host "[1/4] frontend build (type check + bundle)"
Push-Location (Join-Path $repoRoot "frontend")
try {
    if (-not (Test-Path "node_modules")) { npm install }
    npm run build
    if ($LASTEXITCODE -ne 0) { throw "npm run build failed" }
}
finally { Pop-Location }

Write-Host "[2/4] windows exe (embeds the real UI)"
$go = (Get-Command go -ErrorAction SilentlyContinue).Source
if (-not $go) { throw "go not on PATH" }
$env:GOOS = "windows"
try {
    & $go build -o (Join-Path $repoRoot "bin/hypo.exe") $repoRoot
    if ($LASTEXITCODE -ne 0) { throw "go build failed" }
}
finally { Remove-Item Env:GOOS -ErrorAction SilentlyContinue }

Write-Host "[3/4] staging engine runtime"
New-Item -ItemType Directory -Path (Join-Path $stage "engine/bin") -Force | Out-Null
Copy-Item (Join-Path $repoRoot "bin/hypo.exe") $stage
Copy-Item (Join-Path $engineDir "mole.ps1"), (Join-Path $engineDir "VERSION"), (Join-Path $engineDir "LICENSE") (Join-Path $stage "engine")
Copy-Item (Join-Path $engineDir "bin/*.ps1") (Join-Path $stage "engine/bin")
foreach ($tool in "status.exe", "analyze.exe") {
    $built = Join-Path $engineDir "bin/$tool"
    if (-not (Test-Path $built)) {
        throw "engine/bin/$tool missing - cross-build it first: GOOS=windows go build -o bin/$tool ./cmd/$($tool.Substring(0, $tool.Length - 4))"
    }
    Copy-Item $built (Join-Path $stage "engine/bin")
}
Copy-Item (Join-Path $engineDir "lib") (Join-Path $stage "engine/lib") -Recurse

@"
Hypo — Windows deploy package
=============================

Layout (keep as extracted; do not rename engine\ or separate it from hypo.exe):

  hypo-win64\
  ├─ hypo.exe      GUI (double-click to run)
  ├─ README.txt
  └─ engine\       Mole engine (the only runtime dependency)

Requires: Windows 10 / 11 (WebView2 ships with the OS)

Built: $stamp
Hypo commit:  $(git -C $repoRoot rev-parse --short HEAD 2>$null)
Engine commit: $(git -C $engineDir rev-parse --short HEAD 2>$null)

License: GPL-3.0 (see engine\LICENSE; Hypo uses the same license)
"@ | Set-Content -Path (Join-Path $stage "README.txt")

Write-Host "[4/4] zip"
$zip = "$stage.zip"
if (Test-Path $zip) { Remove-Item $zip -Force }
if (Get-Command zip -ErrorAction SilentlyContinue) {
    Push-Location (Split-Path -Parent $stage)
    try { zip -qr $zip (Split-Path -Leaf $stage) } finally { Pop-Location }
}
else {
    Compress-Archive -Path $stage -DestinationPath $zip
}

Write-Host "Done: $zip"
Write-Host "Engine carries clean --json: $((Select-String -Path (Join-Path $stage 'engine/bin/clean.ps1') -Pattern 'MoleJsonOutputMode' -Quiet))"

# Restore the tracked placeholder so a later commit never picks up build
# output; dist/* is gitignored except this one file.
$placeholder = Join-Path $repoRoot "frontend/dist/index.html"
@"
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <title>Hypo</title>
  </head>
  <body>
    <!--
      Placeholder so `go build` / `go vet` work before the frontend is built.
      The real UI is produced by `npm run build` in frontend/ and replaces
      everything in this folder; only this file is tracked.
    -->
    <p>Hypo frontend is not built yet. Run <code>npm install && npm run build</code> inside <code>frontend/</code>.</p>
  </body>
</html>
"@ | Set-Content -Path $placeholder
Write-Host "dist placeholder restored for committing"
