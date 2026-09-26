package enroll

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSubmit_Success(t *testing.T) {

	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that endpoint and method match
		if r.URL.Path != "/api/v1/machines/enroll" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("X-Operator-Token") != "test-op-token" {
			http.Error(w, "missing operator token", http.StatusUnauthorized)
			return
		}

		resp := Response{
			DeviceId:         "dev-1234",
			MachineId:        "mach-5678",
			CertPem:          "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----",
			CertSerialNumber: "a1b2c3d4",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer fakeServer.Close()

	// 2. Call Submit using the test server's URL
	req := Request{
		MachineId:            "mach-5678",
		PlainEnrollmentToken: "test_token",
		PublicKeyPem:         "test_pub_key",
		DeviceSerialNumber:   "WP-DEV-001",
	}

	result, err := Submit(fakeServer.URL, req, "test-op-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3. Assert response fields
	if result.DeviceId != "dev-1234" {
		t.Errorf("expected DeviceId dev-1234, got %s", result.DeviceId)
	}

	if result.CertSerialNumber != "a1b2c3d4" {
		t.Errorf("expected CertSerialNumber a1b2c3d4, got %s", result.CertSerialNumber)
	}
}
