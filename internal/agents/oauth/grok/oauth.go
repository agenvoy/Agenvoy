package oauthGrok

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pardnchiu/go-pkg/filesystem/keychain"
)

func httpClient() *http.Client {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		base = &http.Transport{}
	}
	transport := base.Clone()
	transport.ResponseHeaderTimeout = 15 * time.Second
	return &http.Client{
		Timeout:   10 * time.Minute,
		Transport: transport,
	}
}

func Load() (*StoredToken, error) {
	raw := keychain.Get(tokenKey)
	if raw == "" {
		return nil, nil
	}
	var t StoredToken
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return &t, nil
}

func HasToken() bool {
	return keychain.Get(tokenKey) != ""
}

func ClearToken() error {
	return keychain.Delete(tokenKey)
}

func EnsureFresh(ctx context.Context, token *StoredToken) (*StoredToken, error) {
	if token != nil && !token.expired() {
		return token, nil
	}
	return refresh(ctx, token)
}

func saveToken(t *StoredToken) error {
	raw, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	return keychain.Set(tokenKey, string(raw))
}
