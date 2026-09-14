package tpm

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

type Machine interface {
	GenerateKey() error

	PublicKeyPem() (string, error)

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

func (s *SimulatedMachine) Close() error {
	s.privateKey = nil
	return nil
}
