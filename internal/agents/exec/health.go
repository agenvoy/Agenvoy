package exec

import (
	"context"
	"net/http"
	"strings"
	"time"

	go_pkg_keychain "github.com/pardnchiu/go-pkg/filesystem/keychain"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	"github.com/pardnchiu/agenvoy/internal/agents"
	agentKeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	"github.com/pardnchiu/agenvoy/internal/agents/probe"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

func checkAgentResponsive(ctx context.Context, agent agentTypes.Agent, timeout time.Duration) bool {
	url, apiKey := compatLivenessTarget(agent)
	if url == "" {
		if name := agentName(agent); probe.Supports(probe.Provider(name)) {
			return probe.Alive(ctx, name, timeout)
		}
		return utils.CheckAgentEndpointAlive(ctx, agent, timeout)
	}

	healthCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var headers map[string]string
	if apiKey != "" {
		headers = map[string]string{"Authorization": "Bearer " + apiKey}
	}
	client := &http.Client{Timeout: timeout}
	_, status, err := go_pkg_http.GET[string](healthCtx, client, url, headers)
	return err == nil && status == http.StatusOK
}

func agentName(agent agentTypes.Agent) string {
	if agent == nil {
		return ""
	}
	for name, a := range agents.Registry().Registry {
		if a == agent {
			return name
		}
	}
	return ""
}

func compatLivenessTarget(agent agentTypes.Agent) (string, string) {
	if agent == nil {
		return "", ""
	}

	for name, a := range agents.Registry().Registry {
		if a != agent {
			continue
		}
		instance, ok := agentKeychain.CompatInstance(name)
		if !ok || instance == "" {
			return "", ""
		}
		baseURL := strings.TrimRight(config.GetCompatURL(instance), "/")
		if baseURL == "" {
			return "", ""
		}
		return baseURL + "/models", go_pkg_keychain.Get("COMPAT_" + instance + "_API_KEY")
	}
	return "", ""
}
