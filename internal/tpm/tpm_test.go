package tpm

import (
	"strings"
	"testing"
)

func TestSimulatedMachine_GenerateAndExportPEM(t *testing.T) {
	sim := NewSimulatedMachine()
	defer func(sim *SimulatedMachine) {
		err := sim.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}(sim)

	// 1. Before generation, PublicKeyPem must return an error
	_, err := sim.PublicKeyPem()
	if err == nil {
		t.Fatal("expected error before GenerateKey, got nil")
	}

	// 2. Generate ECDSA P-256 key pair
	if err := sim.GenerateKey(); err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// 3. Export to PEM format
	pemStr, err := sim.PublicKeyPem()
	if err != nil {
		t.Fatalf("PublicKeyPem failed: %v", err)
	}

	// 4. Verify PEM structure
	if !strings.HasPrefix(pemStr, "-----BEGIN PUBLIC KEY-----") {
		t.Errorf("PEM should start with -----BEGIN PUBLIC KEY-----, got: %s", pemStr)
	}

	if !strings.Contains(pemStr, "-----END PUBLIC KEY-----") {
		t.Errorf("PEM should contain -----END PUBLIC KEY-----, got: %s", pemStr)
	}
}
