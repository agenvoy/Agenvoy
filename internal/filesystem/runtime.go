package filesystem

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"
	go_pkg_sandbox "github.com/pardnchiu/go-pkg/sandbox"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/configs"
)

var (
	MaxToolIterations     = 128
	AgentSendTimeoutSec   = 600
	MaxHistoryMessages    = 24
	MaxHistoryBytes       = 5 * 1024 * 1024
	MaxSessionTasks       = runtime.NumCPU() * 4
	MaxSubagentTimeoutMin = 30
	MaxResumeWaitMin      = 60
)

type SensitiveConfig struct {
	Dirs       []string `json:"dirs"`
	Files      []string `json:"files"`
	Prefixes   []string `json:"prefixes"`
	Extensions []string `json:"extensions"`
}

var (
	SensitivePath   SensitiveConfig
	DeniedCommand   []string
	DeniedPath      []string
	NetWhiteList    []string
	ReadOnlyCommand []string
)

const Port = "17989"

type RuntimeLimits struct {
	MaxToolIterations   int `json:"max_tool_iterations,omitempty"`
	AgentSendTimeoutSec int `json:"agent_send_timeout_seconds,omitempty"`
	MaxHistoryMessages  int `json:"max_history_messages,omitempty"`
	MaxHistoryBytes     int `json:"max_history_bytes,omitempty"`
}

func LoadRuntime() error {
	if ConfigPath == "" {
		return fmt.Errorf("filesystem.LoadRuntime: ConfigPath not initialized (call Init first)")
	}

	raw := map[string]json.RawMessage{}
	if go_pkg_filesystem_reader.Exists(ConfigPath) {
		loaded, err := go_pkg_filesystem.ReadJSON[map[string]json.RawMessage](ConfigPath)
		if err != nil {
			return fmt.Errorf("go_pkg_filesystem.ReadJSON: %w", err)
		}
		raw = loaded
	}

	var limits RuntimeLimits
	if data, ok := raw["limits"]; ok && len(data) > 0 {
		if err := json.Unmarshal(data, &limits); err != nil {
			return fmt.Errorf("json.Unmarshal limits: %w", err)
		}
	}

	changed := false

	if limits.MaxToolIterations <= 0 {
		limits.MaxToolIterations = MaxToolIterations
		changed = true
	}
	MaxToolIterations = limits.MaxToolIterations

	if limits.AgentSendTimeoutSec <= 0 {
		limits.AgentSendTimeoutSec = AgentSendTimeoutSec
		changed = true
	}
	AgentSendTimeoutSec = limits.AgentSendTimeoutSec

	if limits.MaxHistoryMessages <= 0 {
		limits.MaxHistoryMessages = MaxHistoryMessages
		changed = true
	}
	MaxHistoryMessages = limits.MaxHistoryMessages

	if limits.MaxHistoryBytes <= 0 {
		limits.MaxHistoryBytes = MaxHistoryBytes
		changed = true
	}
	MaxHistoryBytes = limits.MaxHistoryBytes

	if err := json.Unmarshal(configs.SensitivePath, &SensitivePath); err != nil {
		return fmt.Errorf("embedded sensitive_path: %w", err)
	}
	if data, ok := raw["sensitive_path"]; ok && len(data) > 0 {
		var user SensitiveConfig
		if err := json.Unmarshal(data, &user); err != nil {
			return fmt.Errorf("json.Unmarshal sensitive_path: %w", err)
		}
		SensitivePath.Dirs = merge(SensitivePath.Dirs, user.Dirs)
		SensitivePath.Files = merge(SensitivePath.Files, user.Files)
		SensitivePath.Prefixes = merge(SensitivePath.Prefixes, user.Prefixes)
		SensitivePath.Extensions = merge(SensitivePath.Extensions, user.Extensions)
	}
	for key, note := range legacyKeys {
		if data, ok := raw[key]; ok && len(data) > 0 {
			slog.Warn("config key is no longer read",
				slog.String("key", key),
				slog.String("note", note))
		}
	}

	if data, ok := raw["denied_command"]; ok && len(data) > 0 {
		var user []string
		if err := json.Unmarshal(data, &user); err != nil {
			return fmt.Errorf("json.Unmarshal denied_command: %w", err)
		}
		DeniedCommand = merge(nil, user)
	}

	if data, ok := raw["denied_path"]; ok && len(data) > 0 {
		var user []string
		if err := json.Unmarshal(data, &user); err != nil {
			return fmt.Errorf("json.Unmarshal denied_path: %w", err)
		}
		DeniedPath = normalizeDeniedPath(user)
	}

	if data, ok := raw["net_white_list"]; ok && len(data) > 0 {
		var user []string
		if err := json.Unmarshal(data, &user); err != nil {
			return fmt.Errorf("json.Unmarshal net_white_list: %w", err)
		}
		NetWhiteList = merge(nil, user)
	}

	if err := json.Unmarshal(configs.ReadOnlyCommand, &ReadOnlyCommand); err != nil {
		return fmt.Errorf("embedded read_only_command: %w", err)
	}
	if data, ok := raw["read_only_command"]; ok && len(data) > 0 {
		var user []string
		if err := json.Unmarshal(data, &user); err != nil {
			return fmt.Errorf("json.Unmarshal read_only_command: %w", err)
		}
		ReadOnlyCommand = merge(ReadOnlyCommand, user)
	}

	if err := applySandboxDenied(); err != nil {
		return err
	}

	if err := go_pkg_filesystem.New(go_pkg_filesystem.Policy{
		ExcludeList: configs.ExcludeList,
	}); err != nil {
		slog.Warn("go_pkg_filesystem New",
			slog.String("error", err.Error()))
	}

	if !changed {
		return nil
	}

	limitsRaw, err := json.Marshal(limits)
	if err != nil {
		return fmt.Errorf("json.Marshal limits: %w", err)
	}
	raw["limits"] = limitsRaw
	if err := go_pkg_filesystem.CheckDir(AgenvoyDir, true); err != nil {
		return fmt.Errorf("go_pkg_filesystem.CheckDir: %w", err)
	}
	if err := go_pkg_filesystem.WriteJSON(ConfigPath, raw, false); err != nil {
		return fmt.Errorf("go_pkg_filesystem.WriteJSON: %w", err)
	}
	return nil
}

var legacyKeys = map[string]string{
	"sensitive_map":   "renamed to sensitive_path",
	"white_list":      "removed; commands run unless listed in denied_command",
	"path_white_list": "removed; paths outside $HOME are approved per session",
}

func normalizeDeniedPath(list []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(list))
	root := string(filepath.Separator)
	for _, entry := range list {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !filepath.IsAbs(entry) && entry != "~" && !strings.HasPrefix(entry, "~/") {
			slog.Warn("denied_path entry dropped: needs an absolute path or ~/",
				slog.String("entry", entry))
			continue
		}
		one := go_pkg_utils.AbsPath("", entry)
		if !filepath.IsAbs(one) {
			slog.Warn("denied_path entry dropped: cannot be resolved",
				slog.String("entry", entry))
			continue
		}
		if one == root {
			slog.Warn("denied_path entry dropped: refusing to deny the filesystem root",
				slog.String("entry", entry))
			continue
		}
		if !seen[one] {
			seen[one] = true
			out = append(out, one)
		}

		resolved, err := go_pkg_filesystem.RealPath(one)
		if err != nil || resolved == root || seen[resolved] {
			continue
		}
		seen[resolved] = true
		out = append(out, resolved)
	}
	return out
}

func applySandboxDenied() error {
	if len(DeniedPath) == 0 {
		return nil
	}
	payload := struct {
		Dirs  []string `json:"dirs"`
		Files []string `json:"files"`
	}{}
	for _, one := range DeniedPath {
		if go_pkg_filesystem_reader.IsFile(one) {
			payload.Files = append(payload.Files, one)
			continue
		}
		payload.Dirs = append(payload.Dirs, one)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json.Marshal denied_path: %w", err)
	}
	go_pkg_sandbox.New(raw)
	return nil
}

func merge(base, extra []string) []string {
	seen := make(map[string]bool, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	for _, v := range base {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	for _, v := range extra {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
