package oauthCodex

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
	// ! agenvoy.codex.token will deprecated in v1.*.*
	if raw == "" {
		raw = keychain.Get("agenvoy.codex.token")
	}
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
	// ! agenvoy.codex.token will deprecated in v1.*.*
	return keychain.Get(tokenKey) != "" || keychain.Get("agenvoy.codex.token") != ""
}

func ClearToken() error {
	err := keychain.Delete(tokenKey)
	// ! agenvoy.codex.token will deprecated in v1.*.*
	if legacyErr := keychain.Delete("agenvoy.codex.token"); legacyErr != nil && err == nil {
		err = legacyErr
	}
	return err
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
