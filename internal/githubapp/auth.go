package githubapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AppAuth struct {
	APIURL         string
	AppID          int64
	InstallationID int64
	PrivateKeyPath string
	HTTPClient     *http.Client
}

func (a *AppAuth) readPrivateKey() ([]byte, error) {
	data, err := os.ReadFile(a.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key %s: %w", a.PrivateKeyPath, err)
	}
	return data, nil
}

func (a *AppAuth) CreateJWT(now time.Time) (string, error) {
	keyBytes, err := a.readPrivateKey()
	if err != nil {
		return "", err
	}

	signingKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}

	iat := now.Add(-1 * time.Minute)
	exp := now.Add(9 * time.Minute)

	claims := jwt.RegisteredClaims{
		Issuer:    fmt.Sprint(a.AppID),
		IssuedAt:  jwt.NewNumericDate(iat),
		ExpiresAt: jwt.NewNumericDate(exp),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(signingKey)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

type installTokenResponse struct {
	Token string `json:"token"`
}

func (a *AppAuth) CreateInstallationToken(now time.Time) (string, error) {
	j, err := a.CreateJWT(now)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", a.APIURL, a.InstallationID)

	client := a.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(nil))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+j)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("post access token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var parsed installTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}
	if parsed.Token == "" {
		return "", fmt.Errorf("token field missing in response")
	}
	return parsed.Token, nil
}
