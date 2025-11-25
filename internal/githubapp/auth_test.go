package githubapp

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
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
