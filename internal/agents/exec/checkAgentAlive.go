package exec

import (
	"context"
	"net/http"
	"strings"
	"time"

	go_pkg_keychain "github.com/pardnchiu/go-pkg/filesystem/keychain"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	agentKeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	"github.com/pardnchiu/agenvoy/internal/agents/probe"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

func checkAgentAlive(ctx context.Context, agent agentTypes.Agent, timeout time.Duration) bool {
	if agent == nil {
		return false
	}

	name := agent.Name()
	url, apiKey := getCompatEndpoint(name)
	if url == "" {
		// * not compat provider
		// * check if agent supports probe
		if probe.Supports(probe.Provider(name)) {
			return probe.Alive(ctx, name, timeout)
		}
		// * check if agent has a health endpoint
		return utils.CheckAgentEndpointAlive(ctx, agent, timeout)
	}

	// * is compat provider
	// * check if compat endpoint is alive
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

func getCompatEndpoint(name string) (string, string) {
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
