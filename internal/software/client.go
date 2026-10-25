package software

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

// IssueRequest payload for POST /api/v1/admin/certificates/software
type IssueRequest struct {
	PublisherName string `json:"publisherName"`
	PublicKeyPem  string `json:"publicKeyPem"`
}

// IssueResponse represents the code signing certificate issued by Step-CA
type IssueResponse struct {
	CertPem          string    `json:"certPem"`
	SerialNumber     string    `json:"serialNumber"`
	ThumbprintSha256 string    `json:"thumbprintSha256"`
	NotAfterUtc      time.Time `json:"notAfterUtc"`
}

// ConfigFile represents an optional software-cert.json configuration file
type ConfigFile struct {
	Token         string `json:"token"`
	PublisherName string `json:"publisherName,omitempty"`
	ApiUrl        string `json:"apiUrl,omitempty"`
}

// LoadConfigFile loads software-cert.json if present
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

// Issue calls POST /api/v1/admin/certificates/software with Bearer auth
func Issue(baseUrl string, req IssueRequest, bearerToken string) (*IssueResponse, error) {
	endpoint := strings.TrimSuffix(baseUrl, "/") + "/api/v1/admin/certificates/software"

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal issue request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+bearerToken)
	}

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
		return nil, fmt.Errorf("certificate request rejected (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var issueResp IssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issueResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issueResp, nil
}
