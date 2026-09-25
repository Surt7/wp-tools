package session

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type StartRequest struct {
	ModeCode      string `json:"modeCode"`
	ClientVersion string `json:"clientVersion"`
}

type StartResponse struct {
	SessionID  string `json:"sessionId"`
	ServerTime int64  `json:"serverTime"`
}

type SlotScoreRequest struct {
	SlotNumber int   `json:"slotNumber"`
	Score      int64 `json:"score"`
}

type SubmitScoreRequest struct {
	DeviceID string             `json:"deviceId"`
	Scores   []SlotScoreRequest `json:"scores"`
}

type SubmitScoreResponse struct {
	ScoreEntryIDs []string `json:"scoreEntryIds"`
}

func BuildCanonical(ts int64, nonce, method, path string, bodyBytes []byte) []byte {
	bodyHash := sha256.Sum256(bodyBytes)
	bodyHashHex := hex.EncodeToString(bodyHash[:])

	canonical := fmt.Sprintf("%d.%s.%s.%s.%s", ts, nonce, strings.ToUpper(method), path, bodyHashHex)
	return []byte(canonical)
}

type Signer interface {
	Sign(data []byte) (string, error)
}

func createSecureHttpClient(certPath, keyPath, caPath string) (*http.Client, error) {
	var certs []tls.Certificate
	if certPath != "" && keyPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate/key: %w", err)
		}
		certs = append(certs, cert)
	}

	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = x509.NewCertPool()
	}

	// If a custom root CA is provided (e.g. from step-ca container)
	if caPath != "" {
		caBytes, err := os.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate from %s: %w", caPath, err)
		}
		if !rootCAs.AppendCertsFromPEM(caBytes) {
			return nil, fmt.Errorf("failed to append CA certificate from %s", caPath)
		}
	}

	tlsConfig := &tls.Config{
		Certificates: certs,
		RootCAs:      rootCAs,
		GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			if len(certs) > 0 {
				return &certs[0], nil
			}
			return nil, nil
		},
		// InsecureSkipVerify remains false for maximum security
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 10 * time.Second,
	}, nil
}

func sendSignedRequest(client *http.Client, apiUrl, method, path, deviceID string, signer Signer, reqBody interface{}) (*http.Response, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	ts := time.Now().Unix()
	nonce := fmt.Sprintf("%d-%s", ts, deviceID[:8])

	canonical := BuildCanonical(ts, nonce, method, path, bodyBytes)
	signature, err := signer.Sign(canonical)
	if err != nil {
		return nil, fmt.Errorf("signing failed: %w", err)
	}

	httpReq, err := http.NewRequest(method, apiUrl+path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Device-Id", deviceID)
	httpReq.Header.Set("X-Timestamp", fmt.Sprintf("%d", ts))
	httpReq.Header.Set("X-Nonce", nonce)
	httpReq.Header.Set("X-Signature", signature)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return resp, nil
}

func executeSignedRequest[T any](client *http.Client, apiUrl, method, path, deviceID string, signer Signer, reqBody interface{}) (*T, error) {
	resp, err := sendSignedRequest(client, apiUrl, method, path, deviceID, signer, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected response status: %d, body: %s", resp.StatusCode, string(respBytes))
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func StartSession(apiUrl, deviceID string, signer Signer, modeCode, certPath, keyPath, caPath string) (*StartResponse, error) {
	client, err := createSecureHttpClient(certPath, keyPath, caPath)
	if err != nil {
		return nil, err
	}

	reqBody := StartRequest{
		ModeCode:      modeCode,
		ClientVersion: "1.0.0",
	}

	return executeSignedRequest[StartResponse](client, apiUrl, http.MethodPost, "/api/v1/sessions/start", deviceID, signer, reqBody)
}

func SubmitScore(apiUrl, deviceID string, signer Signer, sessionID string, scores []SlotScoreRequest, certPath, keyPath, caPath string) (*SubmitScoreResponse, error) {
	client, err := createSecureHttpClient(certPath, keyPath, caPath)
	if err != nil {
		return nil, err
	}

	reqBody := SubmitScoreRequest{
		DeviceID: deviceID,
		Scores:   scores,
	}

	path := fmt.Sprintf("/api/v1/sessions/%s/submit-score", sessionID)
	return executeSignedRequest[SubmitScoreResponse](client, apiUrl, http.MethodPost, path, deviceID, signer, reqBody)
}
