package provenance

import (
	"crypto/sha256"
	"encoding/hex"
)

// signerIdentityRegexp matches the keyless signer identity bound to this repo's Actions
// workflows. Used by CosignVerifyCmd; the OIDC issuer pins GitHub.
const (
	signerIdentityRegexp = "^https://github.com/GetEvinced/stark-marketplace/"
	oidcIssuer           = "https://token.actions.githubusercontent.com"
)

// VerifyDigests recomputes sha256 over the provided files and returns the paths whose digest
// does NOT match the manifest (or are missing). Empty slice = match. This is the ANTI-DRIFT
// layer; the cosign signature (CosignVerifyCmd) is provenance.
func VerifyDigests(m *BuildManifest, files map[string][]byte) []string {
	var mismatches []string
	for _, fd := range m.Files {
		data, ok := files[fd.Path]
		if !ok {
			mismatches = append(mismatches, fd.Path)
			continue
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != fd.Digest {
			mismatches = append(mismatches, fd.Path)
		}
	}
	return mismatches
}

// CosignVerifyCmd returns the argv for keyless verification of a signed manifest. stark
// verify-manifest shells out to this (cosign must be on PATH). The signer identity + OIDC issuer
// are pinned so a signature from any other identity fails.
func CosignVerifyCmd(manifest, sig, cert string) []string {
	return []string{
		"cosign", "verify-blob",
		"--certificate-identity-regexp", signerIdentityRegexp,
		"--certificate-oidc-issuer", oidcIssuer,
		"--signature", sig,
		"--certificate", cert,
		manifest,
	}
}
