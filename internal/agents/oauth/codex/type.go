package oauthCodex

const (
	tokenKey    = "CODEX_OAUTH_TOKEN"
	clientID    = "app_EMoamEEZ73f0CkXaXp7hrann"
	authURL     = "https://auth.openai.com/oauth/authorize"
	tokenURL    = "https://auth.openai.com/oauth/token"
	redirectURI = "http://localhost:1455/auth/callback"
	scopes      = "openid email profile offline_access"
)

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	// * Error / ErrorDesc are `any` because OpenAI's token endpoint switched
	// * `error` from string to object (e.g. {"code":"...","message":"..."}).
	Error     any `json:"error,omitempty"`
	ErrorDesc any `json:"error_description,omitempty"`
}
