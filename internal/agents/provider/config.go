package provider

type Config struct {
	Model   string
	APIKey  string
	Token   any
	BaseURL string

	// * cloudflare
	AccountID string
	GatewayID string
}
