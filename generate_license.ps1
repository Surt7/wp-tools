param(
    [string]$HardwareBinding = "poste-admin-02",
    [string]$Product = "Pinball Aegis Enterprise",
    [string]$Edition = "Operator Field Station",
    [string]$LicensedTo = "WorldPinball Operations Europe",
    [string]$CertPath = "D:\Workspace\GoProject\wp-tools\software-signing.crt",
    [string]$KeyPath = "D:\Workspace\GoProject\wp-tools\software-signing.key",
    [string]$OutPath = "D:\Workspace\GoProject\wp-tools\aegis.lic"
)

if (![System.IO.File]::Exists($CertPath) -or ![System.IO.File]::Exists($KeyPath)) {
    Write-Error "Signing cert or key not found: $CertPath / $KeyPath"
    exit 1
}

$cert = [System.Security.Cryptography.X509Certificates.X509Certificate2]::CreateFromPemFile($CertPath, $KeyPath)
$ecdsa = [System.Security.Cryptography.X509Certificates.ECDsaCertificateExtensions]::GetECDsaPrivateKey($cert)
if ($null -eq $ecdsa) {
    Write-Error "Certificate does not have an ECDSA private key"
    exit 1
}

$payloadObj = [PSCustomObject]@{
    licenseId = "LIC-2026-AEGIS-042"
    product = $Product
    edition = $Edition
    licensedTo = $LicensedTo
    hardwareBinding = $HardwareBinding
    issuedAtUtc = (Get-Date).ToUniversalTime().ToString("O")
    expiresAtUtc = (Get-Date).AddYears(1).ToUniversalTime().ToString("O")
    features = @("TpmAttestation", "FleetManagement", "FirmwareDeployment", "DiagnosticTelemetry")
}

$payloadJson = $payloadObj | ConvertTo-Json -Depth 5 -Compress
$payloadBytes = [System.Text.Encoding]::UTF8.GetBytes($payloadJson)
$signatureBytes = $ecdsa.SignData(
    $payloadBytes, 
    [System.Security.Cryptography.HashAlgorithmName]::SHA256, 
    [System.Security.Cryptography.DSASignatureFormat]::Rfc3279DerSequence
)

$envelope = [PSCustomObject]@{
    payloadBase64 = [Convert]::ToBase64String($payloadBytes)
    signatureBase64 = [Convert]::ToBase64String($signatureBytes)
}

$envelope | ConvertTo-Json | Set-Content -Path $OutPath -Encoding UTF8
Write-Output "SUCCESS: Signed cryptographic license created at: $OutPath"
