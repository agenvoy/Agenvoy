package oauthCopilot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/pardnchiu/go-pkg/filesystem/keychain"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"
)

const (
	tokenKey            = "COPILOT_OAUTH_TOKEN"
	deviceCodeAPI       = "https://github.com/login/device/code"
	oauthAccessTokenAPI = "https://github.com/login/oauth/access_token"
	clientID            = "Iv1.b507a08c87ecfe98" // TODO: will replace with personal client id
)

var (
	errAuthorizationPending = fmt.Errorf("authorization pending") // * pre declare error for ensuring padding wont cause login exit
)

type Token struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	Scope       string    `json:"scope"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type GopilotAccessToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

func Load() (*Token, error) {
	raw := keychain.Get(tokenKey)
	// ! agenvoy.copilot.token will deprecated in v1.*.*
	if raw == "" {
		raw = keychain.Get("agenvoy.copilot.token")
	}
	if raw == "" {
		return nil, nil
	}
	var t Token
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return &t, nil
}

func HasToken() bool {
	// ! agenvoy.copilot.token will deprecated in v1.*.*
	return keychain.Get(tokenKey) != "" || keychain.Get("agenvoy.copilot.token") != ""
}

func ClearToken() error {
	err := keychain.Delete(tokenKey)
	// ! agenvoy.copilot.token will deprecated in v1.*.*
	if legacyErr := keychain.Delete("agenvoy.copilot.token"); legacyErr != nil && err == nil {
		err = legacyErr
	}
	return err
}

func LoginWithCallback(ctx context.Context, onCode func(*DeviceCode)) (*Token, error) {
	code, _, err := go_pkg_http.POST[DeviceCode](ctx, nil, deviceCodeAPI,
		map[string]string{},
		map[string]any{
			"client_id": clientID,
		}, "form")
	if err != nil {
		return nil, fmt.Errorf("device-code: %w", err)
	}

	if onCode != nil {
		onCode(&code)
	}

	interval := time.Duration(code.Interval) * time.Second
	deadline := time.Now().Add(time.Duration(code.ExpiresIn) * time.Second)

	var token *Token
	client := &http.Client{Timeout: 30 * time.Second}
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		token, err = getAccessToken(ctx, client, code.DeviceCode)
		if err != nil {
			// * waiting for authorize
			if errors.Is(err, errAuthorizationPending) {
				continue
			}
			return nil, err
		}
		return token, nil
	}
	return nil, fmt.Errorf("device code expired")
}

func getAccessToken(ctx context.Context, client *http.Client, deviceCode string) (*Token, error) {
	accessToken, _, err := go_pkg_http.POST[GopilotAccessToken](ctx, client, oauthAccessTokenAPI,
		map[string]string{},
		map[string]any{
			"client_id":   clientID,
			"device_code": deviceCode,
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		}, "form")
	if err != nil {
		return nil, err
	}

	switch accessToken.Error {
	case "":
		token := &Token{
			AccessToken: accessToken.AccessToken,
			TokenType:   accessToken.TokenType,
			Scope:       accessToken.Scope,
		}

		raw, err := json.Marshal(token)
		if err != nil {
			return nil, fmt.Errorf("json.Marshal: %w", err)
		}
		if err := keychain.Set(tokenKey, string(raw)); err != nil {
			return nil, fmt.Errorf("keychain.Set: %w", err)
		}
		return token, nil

	case "authorization_pending":
		return nil, errAuthorizationPending

	default:
		return nil, fmt.Errorf("accessToken.Error: %s", accessToken.Error)
	}
}
