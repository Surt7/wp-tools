package tpm

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

type Machine interface {
	GenerateKey() error

	PublicKeyPem() (string, error)
	Sign(data []byte) (string, error)
	SaveKeyToFile(filePath string) error
	LoadKeyFromFile(filePath string) error
	Close() error
}

type SimulatedMachine struct {
	privateKey *ecdsa.PrivateKey
}

func NewSimulatedMachine() *SimulatedMachine {
	return &SimulatedMachine{}
}

// GenerateKey creates a software ECDSA P-256 key pair.
func (s *SimulatedMachine) GenerateKey() error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	s.privateKey = priv
	return nil
}

func (s *SimulatedMachine) PublicKeyPem() (string, error) {
	if s.privateKey == nil {
		return "", errors.New("key not generated yet: call GenerateKey first")
	}

	der, err := x509.MarshalPKIXPublicKey(&s.privateKey.PublicKey)
	if err != nil {
		return "", err
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}

	return string(pem.EncodeToMemory(block)), nil

}

func (s *SimulatedMachine) Sign(data []byte) (string, error) {

	if s.privateKey == nil {
		return "", errors.New("private key not initialized")
	}
	hash := sha256.Sum256(data)

	sigDer, err := ecdsa.SignASN1(rand.Reader, s.privateKey, hash[:])
	if err != nil {
		return "", err

	}
	return base64.StdEncoding.EncodeToString(sigDer), nil
}

// SaveKeyToFile saves the private key to disk in PEM format.
func (s *SimulatedMachine) SaveKeyToFile(filePath string) error {
	if s.privateKey == nil {
		return errors.New("tpm: no private key to save")
	}
	der, err := x509.MarshalECPrivateKey(s.privateKey)
	if err != nil {
		return fmt.Errorf("tpm: failed to encode private key: %w", err)
	}
	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	}
	return os.WriteFile(filePath, pem.EncodeToMemory(block), 0600)
}

// LoadKeyFromFile loads an existing private key from a PEM file.
func (s *SimulatedMachine) LoadKeyFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("tpm: failed to read key file: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "EC PRIVATE KEY" {
		return errors.New("tpm: invalid private key format")
	}
	priv, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("tpm: failed to parse private key: %w", err)
	}
	s.privateKey = priv
	return nil
}

func (s *SimulatedMachine) Close() error {
	s.privateKey = nil
	return nil
}
