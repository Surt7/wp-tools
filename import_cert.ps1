param([string]$StoreLocation = "CurrentUser")

$pfxPath = "D:\Workspace\GoProject\wp-tools\workstation.pfx"
if (![System.IO.File]::Exists($pfxPath)) {
    Write-Error "PFX file not found: $pfxPath"
    exit 1
}

$flags = [System.Security.Cryptography.X509Certificates.X509KeyStorageFlags]::PersistKeySet
if ($StoreLocation -eq "LocalMachine") {
    $flags = $flags -bor [System.Security.Cryptography.X509Certificates.X509KeyStorageFlags]::MachineKeySet
} else {
    $flags = $flags -bor [System.Security.Cryptography.X509Certificates.X509KeyStorageFlags]::UserKeySet
}

$pfx = [System.Security.Cryptography.X509Certificates.X509Certificate2]::new(
    $pfxPath,
    "",
    $flags
)

$store = [System.Security.Cryptography.X509Certificates.X509Store]::new("My", $StoreLocation)
try {
    $store.Open("ReadWrite")
    $store.Add($pfx)
    $store.Close()
    Write-Output "SUCCESS: Certificate '$($pfx.Subject)' successfully installed into Cert:\$StoreLocation\My"
} catch {
    Write-Error "Failed to install to $StoreLocation : $($_.Exception.Message)"
}
