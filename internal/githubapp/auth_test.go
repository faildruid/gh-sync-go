package githubapp

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeTransport struct {
	statusCode int
	body       []byte
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := &http.Response{
		StatusCode: f.statusCode,
		Body:       io.NopCloser(bytes.NewReader(f.body)),
		Header:     make(http.Header),
		Request:    req,
	}
	return resp, nil
}

func writeTestKey(t *testing.T, dir string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privBytes := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	pemBytes := pem.EncodeToMemory(block)
	path := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	return path
}

func TestCreateJWTSuccess(t *testing.T) {
	dir := t.TempDir()
	keyPath := writeTestKey(t, dir)

	auth := AppAuth{
		APIURL:         "https://gh/api/v3",
		AppID:          123,
		InstallationID: 456,
		PrivateKeyPath: keyPath,
	}

	now := time.Unix(1_000_000, 0)
	token, err := auth.CreateJWT(now)
	if err != nil {
		t.Fatalf("CreateJWT returned error: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non empty token")
	}
}

func TestCreateJWTMissingKey(t *testing.T) {
	auth := AppAuth{
		APIURL:         "https://gh/api/v3",
		AppID:          1,
		InstallationID: 2,
		PrivateKeyPath: "does-not-exist.pem",
	}
	_, err := auth.CreateJWT(time.Unix(0, 0))
	if err == nil {
		t.Fatalf("expected error for missing key, got nil")
	}
}

func TestCreateInstallationTokenSuccess(t *testing.T) {
	dir := t.TempDir()
	keyPath := writeTestKey(t, dir)

	auth := AppAuth{
		APIURL:         "https://gh/api/v3",
		AppID:          123,
		InstallationID: 999,
		PrivateKeyPath: keyPath,
	}
	client := &http.Client{
		Transport: &fakeTransport{
			statusCode: http.StatusCreated,
			body:       []byte(`{"token":"access_token"}`),
		},
	}
	auth.HTTPClient = client

	token, err := auth.CreateInstallationToken(time.Unix(1_000_000, 0))
	if err != nil {
		t.Fatalf("CreateInstallationToken error: %v", err)
	}
	if token != "access_token" {
		t.Fatalf("unexpected token %q", token)
	}
}

func TestCreateInstallationTokenBadStatus(t *testing.T) {
	dir := t.TempDir()
	keyPath := writeTestKey(t, dir)

	auth := AppAuth{
		APIURL:         "https://gh/api/v3",
		AppID:          123,
		InstallationID: 999,
		PrivateKeyPath: keyPath,
	}
	client := &http.Client{
		Transport: &fakeTransport{
			statusCode: http.StatusInternalServerError,
			body:       []byte("boom"),
		},
	}
	auth.HTTPClient = client

	_, err := auth.CreateInstallationToken(time.Unix(1_000_000, 0))
	if err == nil {
		t.Fatalf("expected error for bad status, got nil")
	}
}

func TestCreateInstallationTokenMissingTokenField(t *testing.T) {
	dir := t.TempDir()
	keyPath := writeTestKey(t, dir)

	auth := AppAuth{
		APIURL:         "https://gh/api/v3",
		AppID:          123,
		InstallationID: 999,
		PrivateKeyPath: keyPath,
	}
	client := &http.Client{
		Transport: &fakeTransport{
			statusCode: http.StatusCreated,
			body:       []byte(`{}`),
		},
	}
	auth.HTTPClient = client

	_, err := auth.CreateInstallationToken(time.Unix(1_000_000, 0))
	if err == nil {
		t.Fatalf("expected error for missing token, got nil")
	}
}

func TestCreateJWTInvalidKeyFormat(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "invalid.pem")
	// Write invalid PEM content
	if err := os.WriteFile(keyPath, []byte("not a valid private key"), 0o600); err != nil {
		t.Fatalf("write invalid key: %v", err)
	}

	auth := AppAuth{
		APIURL:         "https://gh/api/v3",
		AppID:          123,
		InstallationID: 456,
		PrivateKeyPath: keyPath,
	}

	_, err := auth.CreateJWT(time.Unix(1_000_000, 0))
	if err == nil {
		t.Fatalf("expected error for invalid key format, got nil")
	}
}

func TestCreateInstallationTokenHTTPError(t *testing.T) {
	dir := t.TempDir()
	keyPath := writeTestKey(t, dir)

	auth := AppAuth{
		APIURL:         "https://gh/api/v3",
		AppID:          123,
		InstallationID: 999,
		PrivateKeyPath: keyPath,
	}
	client := &http.Client{
		Transport: &fakeTransportWithError{},
	}
	auth.HTTPClient = client

	_, err := auth.CreateInstallationToken(time.Unix(1_000_000, 0))
	if err == nil {
		t.Fatalf("expected error for HTTP error, got nil")
	}
}

func TestCreateInstallationTokenInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	keyPath := writeTestKey(t, dir)

	auth := AppAuth{
		APIURL:         "https://gh/api/v3",
		AppID:          123,
		InstallationID: 999,
		PrivateKeyPath: keyPath,
	}
	client := &http.Client{
		Transport: &fakeTransport{
			statusCode: http.StatusCreated,
			body:       []byte(`invalid json`),
		},
	}
	auth.HTTPClient = client

	_, err := auth.CreateInstallationToken(time.Unix(1_000_000, 0))
	if err == nil {
		t.Fatalf("expected error for invalid JSON, got nil")
	}
}

func TestCreateInstallationTokenJWTError(t *testing.T) {
	auth := AppAuth{
		APIURL:         "https://gh/api/v3",
		AppID:          123,
		InstallationID: 999,
		PrivateKeyPath: "nonexistent.pem",
	}

	_, err := auth.CreateInstallationToken(time.Unix(1_000_000, 0))
	if err == nil {
		t.Fatalf("expected error when JWT creation fails, got nil")
	}
}

func TestCreateInstallationTokenUsesDefaultClient(t *testing.T) {
	dir := t.TempDir()
	keyPath := writeTestKey(t, dir)

	auth := AppAuth{
		APIURL:         "https://gh/api/v3",
		AppID:          123,
		InstallationID: 999,
		PrivateKeyPath: keyPath,
		HTTPClient:     nil, // Test that nil client uses default
	}

	// This will fail because we don't have a real server, but it verifies
	// the code path that uses http.DefaultClient
	_, err := auth.CreateInstallationToken(time.Unix(1_000_000, 0))
	// We expect an error (network error), but the important thing is we
	// didn't panic and the default client code path was executed
	if err == nil {
		t.Fatalf("expected network error, got nil")
	}
}

type fakeTransportWithError struct{}

func (f *fakeTransportWithError) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("network error")
}

func TestReadPrivateKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := writeTestKey(t, dir)

	auth := AppAuth{
		PrivateKeyPath: keyPath,
	}

	data, err := auth.readPrivateKey()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("expected non-empty key data")
	}
}

func TestReadPrivateKeyMissingFile(t *testing.T) {
	auth := AppAuth{
		PrivateKeyPath: "nonexistent.pem",
	}

	_, err := auth.readPrivateKey()
	if err == nil {
		t.Fatalf("expected error for missing file, got nil")
	}
}

func TestCreateInstallationTokenInvalidURL(t *testing.T) {
	dir := t.TempDir()
	keyPath := writeTestKey(t, dir)

	auth := AppAuth{
		APIURL:         "://invalid-url",
		AppID:          123,
		InstallationID: 999,
		PrivateKeyPath: keyPath,
	}
	client := &http.Client{
		Transport: &fakeTransport{
			statusCode: http.StatusCreated,
			body:       []byte(`{"token":"test"}`),
		},
	}
	auth.HTTPClient = client

	_, err := auth.CreateInstallationToken(time.Unix(1_000_000, 0))
	if err == nil {
		t.Fatalf("expected error for invalid URL in request creation, got nil")
	}
}

func TestCreateJWTWithTinyKey(t *testing.T) {
	// Create a very small RSA key (smaller than recommended)
	// This tests edge cases in key handling
	dir := t.TempDir()

	// Generate a tiny key - this might cause issues with signing
	key, err := rsa.GenerateKey(rand.Reader, 512) // Very small key
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privBytes := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	pemBytes := pem.EncodeToMemory(block)
	keyPath := filepath.Join(dir, "tiny.pem")
	if err := os.WriteFile(keyPath, pemBytes, 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	auth := AppAuth{
		APIURL:         "https://gh/api/v3",
		AppID:          123,
		InstallationID: 456,
		PrivateKeyPath: keyPath,
	}

	now := time.Unix(1_000_000, 0)
	token, err := auth.CreateJWT(now)
	// Even with a tiny key, JWT signing should work
	// This test ensures we handle various key sizes
	if err != nil {
		t.Fatalf("CreateJWT with tiny key returned error: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non empty token")
	}
}
