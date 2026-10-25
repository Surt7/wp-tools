package software

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIssue_Success(t *testing.T) {
	expectedSerial := "software-serial-456"
	expectedThumbprint := "software-thumbprint-def"
	expectedNotAfter := time.Now().Add(365 * 24 * time.Hour).UTC().Truncate(time.Second)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/admin/certificates/software" {
			t.Errorf("expected path /api/v1/admin/certificates/software, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-jwt-token" {
			t.Errorf("expected Authorization 'Bearer test-jwt-token', got '%s'", r.Header.Get("Authorization"))
		}

		var req IssueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.PublisherName != "WorldPinball Studio" {
			t.Errorf("expected publisher 'WorldPinball Studio', got '%s'", req.PublisherName)
		}

		resp := IssueResponse{
			CertPem:          "-----BEGIN CERTIFICATE-----\nTEST-CODE-SIGNING\n-----END CERTIFICATE-----",
			SerialNumber:     expectedSerial,
			ThumbprintSha256: expectedThumbprint,
			NotAfterUtc:      expectedNotAfter,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	req := IssueRequest{
		PublisherName: "WorldPinball Studio",
		PublicKeyPem:  "dummy-pubkey",
	}

	resp, err := Issue(server.URL, req, "test-jwt-token")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.SerialNumber != expectedSerial {
		t.Errorf("expected serial %s, got %s", expectedSerial, resp.SerialNumber)
	}
	if resp.ThumbprintSha256 != expectedThumbprint {
		t.Errorf("expected thumbprint %s, got %s", expectedThumbprint, resp.ThumbprintSha256)
	}
}

func TestIssue_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	req := IssueRequest{
		PublisherName: "WorldPinball Studio",
		PublicKeyPem:  "dummy-pubkey",
	}

	resp, err := Issue(server.URL, req, "expired-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}
}

func TestLoadConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-software.json")

	content := `{"token":"jwt-123","publisherName":"Studio","apiUrl":"https://api.test"}`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := LoadConfigFile(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Token != "jwt-123" {
		t.Errorf("expected token 'jwt-123', got '%s'", cfg.Token)
	}
	if cfg.PublisherName != "Studio" {
		t.Errorf("expected publisher 'Studio', got '%s'", cfg.PublisherName)
	}
}
