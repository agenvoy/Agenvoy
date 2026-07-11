package oauthCodex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func refresh(ctx context.Context, token *StoredToken) (*StoredToken, error) {
	if token == nil || token.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {token.RefreshToken},
		"client_id":     {clientID},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("http.NewRequestWithContext: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("httpClient.Do: %w", err)
	}
	defer resp.Body.Close()

	var raw oauthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("json.Decode: %w", err)
	}
	if raw.Error != nil {
		return nil, fmt.Errorf("refresh error %v: %v", raw.Error, raw.ErrorDesc)
	}

	expiry := time.Now().Add(time.Duration(raw.ExpiresIn) * time.Second)
	if raw.ExpiresIn == 0 {
		expiry = time.Now().Add(3600 * time.Second)
	}

	refreshToken := raw.RefreshToken
	if refreshToken == "" {
		refreshToken = token.RefreshToken
	}

	accountID := parseAccountID(raw.IDToken)
	if accountID == "" {
		accountID = token.AccountID
	}

	next := &StoredToken{
		AccessToken:  raw.AccessToken,
		RefreshToken: refreshToken,
		IDToken:      raw.IDToken,
		AccountID:    accountID,
		ExpiresAt:    expiry,
	}

	if err := saveToken(next); err != nil {
		return nil, err
	}
	return next, nil
}
