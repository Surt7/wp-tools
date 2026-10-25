package workstation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnroll_Success(t *testing.T) {
	expectedSerial := "test-serial-123"
	expectedThumbprint := "test-thumbprint-abc"
	expectedNotAfter := time.Now().Add(365 * 24 * time.Hour).UTC().Truncate(time.Second)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/workstations/enroll" {
			t.Errorf("expected path /api/v1/workstations/enroll, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var req EnrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.PlainBootstrapToken != "valid-token" {
			t.Errorf("expected token 'valid-token', got '%s'", req.PlainBootstrapToken)
		}
		if req.WorkstationName != "test-workstation" {
			t.Errorf("expected workstation 'test-workstation', got '%s'", req.WorkstationName)
		}

		resp := EnrollResponse{
			CertPem:          "-----BEGIN CERTIFICATE-----\nTEST\n-----END CERTIFICATE-----",
			SerialNumber:     expectedSerial,
			ThumbprintSha256: expectedThumbprint,
			NotAfterUtc:      expectedNotAfter,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	req := EnrollRequest{
		PlainBootstrapToken: "valid-token",
		WorkstationName:     "test-workstation",
		PublicKeyPem:        "dummy-pubkey",
	}

	resp, err := Enroll(server.URL, req)
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

func TestEnroll_HttpError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"Invalid token format."}`))
	}))
	defer server.Close()

	req := EnrollRequest{
		PlainBootstrapToken: "invalid-token",
		WorkstationName:     "test-workstation",
		PublicKeyPem:        "dummy-pubkey",
	}

	resp, err := Enroll(server.URL, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}
}

func TestLoadConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-enroll.json")

	content := `{"token":"test-token","workstationName":"my-pc","apiUrl":"https://api.test"}`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := LoadConfigFile(configPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Token != "test-token" {
		t.Errorf("expected token 'test-token', got '%s'", cfg.Token)
	}
	if cfg.WorkstationName != "my-pc" {
		t.Errorf("expected workstation 'my-pc', got '%s'", cfg.WorkstationName)
	}
	if cfg.ApiUrl != "https://api.test" {
		t.Errorf("expected apiUrl 'https://api.test', got '%s'", cfg.ApiUrl)
	}
}
