package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/agents/probe"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	imageTool "github.com/pardnchiu/agenvoy/internal/tools/external/image"
	oauthCodex "github.com/pardnchiu/go-llm-router/core/oauth/codex"
	oauthCopilot "github.com/pardnchiu/go-llm-router/core/oauth/copilot"
	oauthGrokOauth "github.com/pardnchiu/go-llm-router/core/oauth/grok"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
)

type providerInfo struct {
	ID      string            `json:"id"`
	Label   string            `json:"label"`
	Methods map[string]string `json:"methods"`
}

type providerState struct {
	providerInfo
	LoggedIn bool `json:"logged_in"`
}

var providerCatalog = []providerInfo{
	{"openai", "OpenAI", map[string]string{"api_key": "pay per token"}},
	{"codex", "OpenAI Codex", map[string]string{"oauth": "Codex subscription"}},
	{"claude", "Claude", map[string]string{"api_key": "pay per token"}},
	{"gemini", "Gemini", map[string]string{"api_key": "pay per token"}},
	{"grok", "Grok", map[string]string{"api_key": "pay per token"}},
	{"grok-oauth", "Grok (xAI)", map[string]string{"oauth": "xAI subscription"}},
	{"copilot", "GitHub Copilot", map[string]string{"oauth": "GitHub subscription"}},
	{"deepseek", "DeepSeek", map[string]string{"api_key": "pay per token"}},
	{"mistral", "Mistral", map[string]string{"api_key": "pay per token"}},
	{"nvidia", "NVIDIA NIM", map[string]string{"api_key": "pay per token"}},
	{"openrouter", "OpenRouter", map[string]string{"api_key": "pay per token"}},
	{"cloudflare", "Cloudflare", map[string]string{"api_key": "Workers AI · API token + account ID"}},
	{"compat", "Local/Custom", map[string]string{"custom": "Ollama, LM Studio, or custom URL"}},
}

func findProvider(id string) *providerInfo {
	for i := range providerCatalog {
		if providerCatalog[i].ID == id {
			return &providerCatalog[i]
		}
	}
	return nil
}

func providerLoggedIn(id string) bool {
	switch id {
	case "codex":
		return oauthCodex.HasToken()
	case "copilot":
		return oauthCopilot.HasToken()
	case "grok-oauth":
		return oauthGrokOauth.HasToken()
	}
	return false
}

func ListProviders() gin.HandlerFunc {
	return func(c *gin.Context) {
		list := make([]providerState, 0, len(providerCatalog))
		for _, provider := range providerCatalog {
			list = append(list, providerState{
				providerInfo: provider,
				LoggedIn:     providerLoggedIn(provider.ID),
			})
		}
		c.JSON(http.StatusOK, gin.H{"providers": list})
	}
}

func AddProviderKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		prov := c.Param("provider")

		if prov == "compat" {
			var body struct {
				Name   string `json:"name"`
				URL    string `json:"url"`
				APIKey string `json:"api_key"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			name := strings.TrimSpace(body.Name)
			url := strings.TrimSpace(body.URL)
			if name == "" || url == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "name and url are required"})
				return
			}
			envKey := "COMPAT_API_KEY"
			if name != "" {
				envKey = "COMPAT_" + strings.ToUpper(name) + "_API_KEY"
			}
			if key := strings.TrimSpace(body.APIKey); key != "" {
				if err := keychain.Set(envKey, key); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				if err := config.SaveKey(envKey); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
			if err := config.UpsertCompat(name, url); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			DropQuotaCache(prov)
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}

		p := findProvider(prov)
		if p == nil || p.Methods["api_key"] == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "provider does not support an API key"})
			return
		}

		var body struct {
			APIKey    string `json:"api_key"`
			AccountID string `json:"account_id"`
			GatewayID string `json:"gateway_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		key := strings.TrimSpace(body.APIKey)
		if key == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "api_key is required"})
			return
		}

		accountID := strings.TrimSpace(body.AccountID)
		if prov == "cloudflare" && accountID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "account_id is required for cloudflare"})
			return
		}

		envKey := strings.ToUpper(prov) + "_API_KEY"
		if err := keychain.Set(envKey, key); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := config.SaveKey(envKey); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		DropQuotaCache(prov)

		if prov == "cloudflare" {
			if err := keychain.Set("CLOUDFLARE_ACCOUNT_ID", accountID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if gatewayID := strings.TrimSpace(body.GatewayID); gatewayID != "" {
				if err := keychain.Set("CLOUDFLARE_GATEWAY_ID", gatewayID); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func ProviderOAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		prov := c.Param("provider")
		p := findProvider(prov)
		if p == nil || p.Methods["oauth"] == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "provider does not support oauth"})
			return
		}

		h := c.Writer.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		c.Writer.WriteHeader(http.StatusOK)
		c.Writer.Flush()

		emit := func(v any) {
			raw, err := json.Marshal(v)
			if err != nil {
				return
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", raw)
			c.Writer.Flush()
		}

		ctx := c.Request.Context()
		var err error
		switch prov {
		case "copilot":
			_, err = oauthCopilot.LoginWithCallback(ctx, func(code *oauthCopilot.DeviceCode) {
				emit(gin.H{"url": code.VerificationURI, "user_code": code.UserCode})
			})
		case "codex":
			_, err = oauthCodex.LoginWithCallback(ctx, func(url string) {
				emit(gin.H{"url": url})
			})
		case "grok-oauth":
			_, err = oauthGrokOauth.LoginWithCallback(ctx, func(url string) {
				emit(gin.H{"url": url})
			})
		}
		if err != nil {
			emit(gin.H{"done": true, "ok": false, "error": err.Error()})
			return
		}
		DropQuotaCache(prov)
		emit(gin.H{"done": true, "ok": true})
	}
}

func ClearProviderOAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		prov := c.Param("provider")
		p := findProvider(prov)
		if p == nil || p.Methods["oauth"] == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "provider does not support oauth"})
			return
		}

		var err error
		switch prov {
		case "codex":
			err = oauthCodex.ClearToken()
		case "copilot":
			err = oauthCopilot.ClearToken()
		case "grok-oauth":
			err = oauthGrokOauth.ClearToken()
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		DropQuotaCache(prov)
		imageTool.Prune(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func listModelsFor(c *gin.Context, credentialName string) {
	if !probe.Supports(probe.Provider(credentialName)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider does not support model listing"})
		return
	}
	ids, err := probe.Models(c.Request.Context(), credentialName)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"models": ids})
}

func ListProviderModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		prov := c.Param("provider")
		if prov == "compat" {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "compat model listing isn't wired up yet; register the model name manually via POST /v1/models"})
			return
		}
		listModelsFor(c, prov)
	}
}
