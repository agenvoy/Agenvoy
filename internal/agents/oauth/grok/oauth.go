package oauthGrok

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pardnchiu/go-pkg/filesystem/keychain"

	"github.com/pardnchiu/agenvoy/internal/agents/provider"
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

func Load() (*provider.GrokToken, error) {
	raw := keychain.Get(tokenKey)
	// ! agenvoy.grok-oauth.token will deprecated in v1.*.*
	if raw == "" {
		raw = keychain.Get("agenvoy.grok-oauth.token")
	}
	if raw == "" {
		return nil, nil
	}
	var t provider.GrokToken
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return &t, nil
}

func HasToken() bool {
	// ! agenvoy.grok-oauth.token will deprecated in v1.*.*
	return keychain.Get(tokenKey) != "" || keychain.Get("agenvoy.grok-oauth.token") != ""
}

func ClearToken() error {
	err := keychain.Delete(tokenKey)
	// ! agenvoy.grok-oauth.token will deprecated in v1.*.*
	if legacyErr := keychain.Delete("agenvoy.grok-oauth.token"); legacyErr != nil && err == nil {
		err = legacyErr
	}
	return err
}

func EnsureFresh(ctx context.Context, token *provider.GrokToken) (*provider.GrokToken, error) {
	if token != nil && !token.Expired() {
		return token, nil
	}
	return refresh(ctx, token)
}

func saveToken(t *provider.GrokToken) error {
	raw, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("json.Marshal: %w", err)
	}
	return keychain.Set(tokenKey, string(raw))
}
