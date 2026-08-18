package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadRSAPrivateKeyFromFile loads an RSA private key from a PEM file.
// The file path must be a trusted, admin-configured path (not user input).
func LoadRSAPrivateKeyFromFile(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path) // #nosec
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	return ParseRSAPrivateKey(keyData)
}

// LoadRSAPublicKeyFromFile loads an RSA public key from a PEM file.
// The file path must be a trusted, admin-configured path (not user input).
func LoadRSAPublicKeyFromFile(path string) (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile(path) // #nosec
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	return ParseRSAPublicKey(keyData)
}

// ParseRSAPrivateKey parses an RSA private key from PEM encoded data
func ParseRSAPrivateKey(keyData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing private key")
	}

	// Try PKCS1 format first
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return privateKey, nil
	}

	// Try PKCS8 format
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	privateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA private key")
	}

	return privateKey, nil
}

// ParseRSAPublicKey parses an RSA public key from PEM encoded data
func ParseRSAPublicKey(keyData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}

	// Try PKCS1 format first
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err == nil {
		return publicKey, nil
	}

	// Try PKIX format
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	publicKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA public key")
	}

	return publicKey, nil
}
