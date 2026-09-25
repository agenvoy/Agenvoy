package subagent

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/session"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

var reasoningLevels = func() []string {
	out := make([]string, 0, int(provider.ReasoningMax)+1)
	for r := provider.ReasoningNone; r <= provider.ReasoningMax; r++ {
		out = append(out, r.String())
	}
	return out
}()

type invokeParams struct {
	Mode         string   `json:"mode,omitempty"`
	Task         string   `json:"task"`
	ReportName   string   `json:"report_name,omitempty"`
	SelfID       string   `json:"self_id,omitempty"`
	Model        string   `json:"model,omitempty"`
	Reasoning    string   `json:"reasoning,omitempty"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
	New          *bool    `json:"new,omitempty"`
	ExcludeTools []string `json:"exclude_tools,omitempty"`
}

func invokeSubagent(ctx context.Context, e *toolTypes.Executor, params invokeParams) (string, error) {
	task := strings.TrimSpace(params.Task)
	if task == "" {
		return "", fmt.Errorf("task is required when mode=invoke")
	}

	sessionID := ""
	if selfID := strings.TrimSpace(params.SelfID); selfID != "" {
		sessionID = session.GetSessionIDBySelfID(selfID)
	}

	model := strings.TrimSpace(params.Model)
	if model != "" {
		if err := checkLegModel(model); err != nil {
			return "", err
		}
	}

	reasoning := strings.TrimSpace(params.Reasoning)
	if _, ok := provider.ParseReasoning(reasoning); !ok {
		reasoning = provider.ReasoningLow.String()
	}

	excludeTools := params.ExcludeTools
	if excludeTools == nil {
		excludeTools = []string{}
	}

	ignoreHistory := sessionID == ""
	if params.New != nil {
		ignoreHistory = *params.New
	}

	output, err := exec.ExecWithSubagent(ctx, task, sessionID, model, reasoning,
		strings.TrimSpace(params.SystemPrompt), excludeTools, e.SessionID, ignoreHistory)
	if err != nil {
		return "", err
	}
	return saveReport(output, params.ReportName)
}

func saveReport(output, reportName string) (string, error) {
	header, body, _ := strings.Cut(output, "\n")

	name := reportSlug(reportName)
	if name == "" {
		name = go_pkg_utils.UUID()[:8]
	}
	path := filepath.Join(filesystem.DownloadDir, "temp-"+name+".md")
	if go_pkg_filesystem_reader.Exists(path) {
		path = filepath.Join(filesystem.DownloadDir, "temp-"+name+"-"+go_pkg_utils.UUID()[:8]+".md")
	}

	if err := go_pkg_filesystem.WriteFile(path, body+"\n", 0644); err != nil {
		return "", fmt.Errorf("go_pkg_filesystem.WriteFile %s: %w", path, err)
	}
	return fmt.Sprintf("%s\nreport: %s\nThe leg's full report is in that file — read_files it before relaying or synthesizing.", header, path), nil
}

func reportSlug(raw string) string {
	slug := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			return unicode.ToLower(r)
		}
		return '-'
	}, strings.TrimSpace(raw))
	slug = strings.Trim(slug, "-")
	if runes := []rune(slug); len(runes) > 64 {
		slug = strings.TrimRight(string(runes[:64]), "-")
	}
	return slug
}

func checkLegModel(model string) error {
	registry := agents.Registry()
	tags := map[string]string{}
	if cfg, err := config.Load(); err == nil {
		tags = cfg.ModelTag
	}

	allowed := make([]string, 0, len(registry.Entries))
	for _, e := range registry.Entries {
		if tags[e.Name] != config.ModelTagPass {
			allowed = append(allowed, e.Name)
		}
	}
	if slices.Contains(allowed, model) {
		return nil
	}

	reason := "is not registered"
	if _, ok := registry.Registry[model]; ok {
		reason = "is pass tier and cannot run a subagent leg"
	}
	return fmt.Errorf("model %q %s; available: %s", model, reason, strings.Join(allowed, ", "))
}
