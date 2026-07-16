package oauthGrok

const (
	tokenKey    = "GROK_OAUTH_TOKEN"
	clientID    = "b1a00492-073a-47ea-816f-4c329264a828"
	authURL     = "https://auth.x.ai/oauth2/authorize"
	tokenURL    = "https://auth.x.ai/oauth2/token"
	redirectURI = "http://127.0.0.1:56121/callback"
	scopes      = "openid profile email offline_access grok-cli:access api:access"
)

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Error        any    `json:"error,omitempty"`
	ErrorDesc    any    `json:"error_description,omitempty"`
}
