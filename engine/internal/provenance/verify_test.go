package provenance

import "testing"

func TestVerifyDigestsMatch(t *testing.T) {
	files := map[string][]byte{"index.json": []byte("hello"), "a.json": []byte("a")}
	m := Compute(map[string]int{}, files)
	mismatches := VerifyDigests(m, files)
	if len(mismatches) != 0 {
		t.Fatalf("expected no mismatches, got %+v", mismatches)
	}
}

func TestVerifyDigestsDetectsTamper(t *testing.T) {
	files := map[string][]byte{"index.json": []byte("hello")}
	m := Compute(map[string]int{}, files)
	tampered := map[string][]byte{"index.json": []byte("HELLO-tampered")}
	mismatches := VerifyDigests(m, tampered)
	if len(mismatches) != 1 || mismatches[0] != "index.json" {
		t.Fatalf("expected index.json mismatch, got %+v", mismatches)
	}
}

func TestVerifyDigestsDetectsMissing(t *testing.T) {
	files := map[string][]byte{"index.json": []byte("hello")}
	m := Compute(map[string]int{}, files)
	mismatches := VerifyDigests(m, map[string][]byte{}) // file gone
	if len(mismatches) != 1 || mismatches[0] != "index.json" {
		t.Fatalf("expected missing-file mismatch, got %+v", mismatches)
	}
}

func TestCosignVerifyCmd(t *testing.T) {
	c := CosignVerifyCmd("m.json", "m.json.sig", "m.json.pem")
	joined := ""
	for _, a := range c {
		joined += a + " "
	}
	for _, want := range []string{
		"cosign", "verify-blob",
		"--certificate-identity-regexp", "GetEvinced/stark-marketplace",
		"--certificate-oidc-issuer", "token.actions.githubusercontent.com",
		"--signature", "m.json.sig", "--certificate", "m.json.pem", "m.json",
	} {
		if !contains(joined, want) {
			t.Fatalf("cosign cmd missing %q: %s", want, joined)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
