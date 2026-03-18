Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "[bootstrap] installing pinned tools"
make init

Write-Host "[bootstrap] generating code"
make generate

Write-Host "[bootstrap] running tests"
make test
