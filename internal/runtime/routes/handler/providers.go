package handler

import (
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	oauthCodex "github.com/pardnchiu/go-llm-router/core/oauth/codex"
	oauthCopilot "github.com/pardnchiu/go-llm-router/core/oauth/copilot"
	oauthGrokOauth "github.com/pardnchiu/go-llm-router/core/oauth/grok"
)

type providerInfo struct {
	ID      string            `json:"id"`
	Label   string            `json:"label"`
	Methods map[string]string `json:"methods"`
}

type providerState struct {
	providerInfo
	LoggedIn bool     `json:"logged_in"`
	Console  []string `json:"console"`
	Hidden   bool     `json:"hidden"`
}

var providerConsole = map[string]map[string]string{
	"openai":       {"key": "https://platform.openai.com/api-keys", "billing": "https://platform.openai.com/settings/organization/billing"},
	"claude":       {"key": "https://platform.claude.com/settings/keys", "billing": "https://console.anthropic.com/settings/billing"},
	"gemini":       {"key": "https://aistudio.google.com/apikey", "billing": "https://aistudio.google.com/apikey"},
	"grok":         {"key": "https://console.x.ai/", "billing": "https://console.x.ai/"},
	"deepseek":     {"key": "https://platform.deepseek.com/api_keys", "billing": "https://platform.deepseek.com/top_up"},
	"mistral":      {"key": "https://console.mistral.ai/api-keys", "billing": "https://console.mistral.ai/billing"},
	"nvidia":       {"key": "https://build.nvidia.com/settings/api-keys", "billing": "https://build.nvidia.com/settings/api-keys"},
	"ollama-cloud": {"key": "https://ollama.com/settings/keys", "billing": "https://ollama.com/settings/keys"},
	"openrouter":   {"key": "https://openrouter.ai/settings/keys", "billing": "https://openrouter.ai/credits"},
	"cloudflare":   {"key": "https://dash.cloudflare.com/profile/api-tokens", "billing": "https://dash.cloudflare.com/profile/api-tokens"},
	"claude-code":  {"plan": "https://claude.com/pricing"},
	"codex":        {"plan": "https://chatgpt.com/pricing"},
	"grok-oauth":   {"plan": "https://grok.com/plans"},
	"copilot":      {"plan": "https://github.com/features/copilot/plans"},
}

var providerCatalog = []providerInfo{
	{"openai", "OpenAI", map[string]string{"api_key": "pay per token"}},
	{"codex", "OpenAI Codex", map[string]string{"oauth": "Codex subscription"}},
	{"claude", "Claude", map[string]string{"api_key": "pay per token"}},
	{"claude-code", "Claude Code", map[string]string{"oauth": "Claude subscription via claude -p"}},
	{"gemini", "Gemini", map[string]string{"api_key": "pay per token"}},
	{"grok", "Grok", map[string]string{"api_key": "pay per token"}},
	{"grok-oauth", "Grok (xAI)", map[string]string{"oauth": "xAI subscription"}},
	{"copilot", "GitHub Copilot", map[string]string{"oauth": "GitHub subscription"}},
	{"deepseek", "DeepSeek", map[string]string{"api_key": "pay per token"}},
	{"mistral", "Mistral", map[string]string{"api_key": "pay per token"}},
	{"nvidia", "NVIDIA NIM", map[string]string{"api_key": "pay per token"}},
	{"ollama-cloud", "Ollama Cloud", map[string]string{"api_key": "API key"}},
	{"openrouter", "OpenRouter", map[string]string{"api_key": "pay per token"}},
	{"cloudflare", "Cloudflare", map[string]string{"api_key": "Workers AI · API token + account ID"}},
	{"compat", "Local/Custom", map[string]string{"custom": "Ollama, LM Studio, or custom URL"}},
}

func providerLoggedIn(id string) bool {
	switch id {
	case "claude-code":
		return agentTypes.ClaudeCodeEnabled()
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
				Console:      slices.Sorted(maps.Keys(providerConsole[provider.ID])),
				Hidden:       provider.ID == "claude-code" && !agentTypes.ClaudeCodeEnabled(),
			})
		}
		c.JSON(http.StatusOK, gin.H{"providers": list})
	}
}

func ProviderConsole() gin.HandlerFunc {
	return func(c *gin.Context) {
		dic := providerConsole[c.Param("provider")]
		page := c.Query("page")
		if page == "" {
			page = "key"
			if _, ok := dic["plan"]; ok {
				page = "plan"
			}
		}
		link, ok := dic[page]
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("no %s page for provider %q", page, c.Param("provider"))})
			return
		}
		c.Redirect(http.StatusFound, link)
	}
}
