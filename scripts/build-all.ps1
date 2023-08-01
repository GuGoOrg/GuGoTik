Write-Host "Please Run Me on the root dir, not in scripts dir."

if (Test-Path output) {
    Write-Host "Output dir existed, deleting and recreating..."
    Remove-Item -Recurse -Force output
}
New-Item -ItemType Directory -Path output\services | Out-Null

Set-Location src\services

$directories = Get-ChildItem -Directory | Where-Object { $_.Name -ne 'health' }

foreach ($dir in $directories) {
    $capitalizedName = $dir.Name.Substring(0, 1).ToUpper() + $dir.Name.Substring(1)

    Set-Location -Path $dir.FullName
    & go build -o "../../../output/services/$($dir.Name)/$($capitalizedName)Service.exe"
    Set-Location -Path ".."
}

Set-Location "..\.."

New-Item -ItemType Directory -Path output\gateway | Out-Null

Set-Location src\web
& go build -o "../../output/gateway/GateWay.exe"
Set-Location "..\.."

Write-Host "OK!"
