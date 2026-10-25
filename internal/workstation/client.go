package workstation

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ConfigFile represents a local workstation-enroll.json provisioning file.
type ConfigFile struct {
	Token           string `json:"token"`
	WorkstationName string `json:"workstationName,omitempty"`
	ApiUrl          string `json:"apiUrl,omitempty"`
}

// LoadConfigFile reads and unmarshals a JSON provisioning file.
func LoadConfigFile(path string) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// EnrollRequest represents the payload for POST /api/v1/workstations/enroll
type EnrollRequest struct {
	PlainBootstrapToken string `json:"plainBootstrapToken"`
	WorkstationName     string `json:"workstationName"`
	PublicKeyPem        string `json:"publicKeyPem"`
}

// EnrollResponse represents the certificate issued by the CA
type EnrollResponse struct {
	CertPem          string    `json:"certPem"`
	SerialNumber     string    `json:"serialNumber"`
	ThumbprintSha256 string    `json:"thumbprintSha256"`
	NotAfterUtc      time.Time `json:"notAfterUtc"`
}

func Enroll(baseUrl string, req EnrollRequest) (*EnrollResponse, error) {
	endpoint := strings.TrimSuffix(baseUrl, "/") + "/api/v1/workstations/enroll"
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal enroll request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Allow self-signed or local certificates in dev environment
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: strings.Contains(baseUrl, "worldpinball.local") || strings.Contains(baseUrl, "localhost"),
		},
	}
	client := &http.Client{Transport: tr, Timeout: 15 * time.Second}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("enrollment rejected (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var enrollResp EnrollResponse
	if err := json.NewDecoder(resp.Body).Decode(&enrollResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &enrollResp, nil
}
