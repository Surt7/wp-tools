package enroll

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type ConfigFile struct {
	MachineId          string `json:"machineId"`
	Token              string `json:"token"`
	DeviceSerialNumber string `json:"deviceSerialNumber,omitempty"`
	ApiUrl             string `json:"apiUrl,omitempty"`
}
type Request struct {
	MachineId            string `json:"machineId"`
	PlainEnrollmentToken string `json:"plainEnrollmentToken"`
	PublicKeyPem         string `json:"publicKeyPem"`
	DeviceSerialNumber   string `json:"deviceSerialNumber"`
}

type Response struct {
	DeviceId         string `json:"deviceId"`
	MachineId        string `json:"machineId"`
	CertPem          string `json:"certPem"`
	CertSerialNumber string `json:"certSerialNumber"`
}

func Submit(baseUrl string, req Request) (*Response, error) {
	endpoint := strings.TrimSuffix(baseUrl, "/") + "/api/v1/machines/enroll"
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("enrollment failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var enrollResp Response
	if err := json.NewDecoder(resp.Body).Decode(&enrollResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &enrollResp, nil
}
func LoadConfigFile(path string) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid json in config file: %w", err)
	}

	return &cfg, nil
}
