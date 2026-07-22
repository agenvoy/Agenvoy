package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	agentKeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	"github.com/pardnchiu/agenvoy/internal/runtime/kuradb"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	provider "github.com/pardnchiu/go-llm-router/core"
	"github.com/pardnchiu/go-llm-router/core/claude"
	"github.com/pardnchiu/go-llm-router/core/cloudflare"
	"github.com/pardnchiu/go-llm-router/core/copilot"
	"github.com/pardnchiu/go-llm-router/core/deepseek"
	"github.com/pardnchiu/go-llm-router/core/gemini"
	"github.com/pardnchiu/go-llm-router/core/grok"
	grokoauth "github.com/pardnchiu/go-llm-router/core/grokOauth"
	"github.com/pardnchiu/go-llm-router/core/nvidia"
	oauthCodex "github.com/pardnchiu/go-llm-router/core/oauth/codex"
	oauthCopilot "github.com/pardnchiu/go-llm-router/core/oauth/copilot"
	oauthGrokOauth "github.com/pardnchiu/go-llm-router/core/oauth/grok"
	openrouter "github.com/pardnchiu/go-llm-router/core/openRouter"
	"github.com/pardnchiu/go-llm-router/core/openai"
	openaicodex "github.com/pardnchiu/go-llm-router/core/openaiCodex"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
)

type providerInfo struct {
	ID      string            `json:"id"`
	Label   string            `json:"label"`
	Methods map[string]string `json:"methods"`
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

func ListProviders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"providers": providerCatalog})
	}
}

func CheckProviderKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		prov := c.Param("provider")
		p := findProvider(prov)
		if p == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown provider"})
			return
		}
		switch {
		case p.Methods["oauth"] != "":
			var exists bool
			switch prov {
			case "codex":
				exists = oauthCodex.HasToken()
			case "copilot":
				exists = oauthCopilot.HasToken()
			case "grok-oauth":
				exists = oauthGrokOauth.HasToken()
			}
			c.JSON(http.StatusOK, gin.H{"exists": exists})
		case p.Methods["api_key"] != "":
			c.JSON(http.StatusOK, gin.H{"exists": keychain.Get(strings.ToUpper(prov)+"_API_KEY") != ""})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "provider does not support a key/oauth check"})
		}
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
		if envKey == "OPENAI_API_KEY" {
			if err := kuradb.SyncOpenAIKey(key); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

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
			if oauthCopilot.HasToken() {
				if cerr := oauthCopilot.ClearToken(); cerr != nil {
					emit(gin.H{"error": cerr.Error()})
					return
				}
			}
			_, err = oauthCopilot.LoginWithCallback(ctx, func(code *oauthCopilot.DeviceCode) {
				emit(gin.H{"url": code.VerificationURI, "user_code": code.UserCode})
			})
		case "codex":
			if oauthCodex.HasToken() {
				if cerr := oauthCodex.ClearToken(); cerr != nil {
					emit(gin.H{"error": cerr.Error()})
					return
				}
			}
			_, err = oauthCodex.LoginWithCallback(ctx, func(url string) {
				emit(gin.H{"url": url})
			})
		case "grok-oauth":
			if oauthGrokOauth.HasToken() {
				if cerr := oauthGrokOauth.ClearToken(); cerr != nil {
					emit(gin.H{"error": cerr.Error()})
					return
				}
			}
			_, err = oauthGrokOauth.LoginWithCallback(ctx, func(url string) {
				emit(gin.H{"url": url})
			})
		}
		if err != nil {
			emit(gin.H{"done": true, "ok": false, "error": err.Error()})
			return
		}
		emit(gin.H{"done": true, "ok": true})
	}
}

func modelsFn(prov string) func(c *gin.Context, cfg provider.Config) ([]string, error) {
	switch prov {
	case "openai":
		return func(c *gin.Context, cfg provider.Config) ([]string, error) {
			return openai.Models(c.Request.Context(), cfg)
		}
	case "codex":
		return func(c *gin.Context, cfg provider.Config) ([]string, error) {
			return openaicodex.Models(c.Request.Context(), cfg)
		}
	case "claude":
		return func(c *gin.Context, cfg provider.Config) ([]string, error) {
			return claude.Models(c.Request.Context(), cfg)
		}
	case "gemini":
		return func(c *gin.Context, cfg provider.Config) ([]string, error) {
			return gemini.Models(c.Request.Context(), cfg)
		}
	case "grok":
		return func(c *gin.Context, cfg provider.Config) ([]string, error) {
			return grok.Models(c.Request.Context(), cfg)
		}
	case "grok-oauth":
		return func(c *gin.Context, cfg provider.Config) ([]string, error) {
			return grokoauth.Models(c.Request.Context(), cfg)
		}
	case "copilot":
		return func(c *gin.Context, cfg provider.Config) ([]string, error) {
			return copilot.Models(c.Request.Context(), cfg)
		}
	case "deepseek":
		return func(c *gin.Context, cfg provider.Config) ([]string, error) {
			return deepseek.Models(c.Request.Context(), cfg)
		}
	case "nvidia":
		return func(c *gin.Context, cfg provider.Config) ([]string, error) {
			return nvidia.Models(c.Request.Context(), cfg)
		}
	case "openrouter":
		return func(c *gin.Context, cfg provider.Config) ([]string, error) {
			return openrouter.Models(c.Request.Context(), cfg)
		}
	case "cloudflare":
		return func(c *gin.Context, cfg provider.Config) ([]string, error) {
			return cloudflare.Models(c.Request.Context(), cfg)
		}
	default:
		return nil
	}
}

func listModelsFor(c *gin.Context, credentialName string) {
	fn := modelsFn(credentialName)
	if fn == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider does not support model listing"})
		return
	}
	cfg, err := agentKeychain.Config(c.Request.Context(), credentialName)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	ids, err := fn(c, cfg)
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
