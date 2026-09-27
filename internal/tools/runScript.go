package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pardnchiu/go-pkg/filesystem/keychain"
	go_pkg_sandbox "github.com/pardnchiu/go-pkg/sandbox"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

const runScriptTimeout = 15 * time.Minute

type scriptRuntime struct {
	binary string
	suffix string
}

var scriptRuntimeMap = map[string]scriptRuntime{
	"python": {binary: "python3", suffix: ".py"},
	"shell":  {binary: "bash", suffix: ".sh"},
}

func registRunScript() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "run_script",
		Timeout:     runScriptTimeout,
		SystemUse:   false,
		AlwaysLoad:  false,
		AlwaysAllow: false,
		Concurrent:  false,
		Description: `Runs a Python or shell script with network access, returns its combined stdout/stderr.
Use for 打 API / 抓資料 / 爬蟲 / 呼叫外部服務 / 寫個腳本跑 — anything reaching a host over more than one request, or doing work between requests. run_command has no network, so every outbound call lands here.
One URL or one JSON endpoint → fetch_page or http_request. Local files only, or any build, test or git work → run_command.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"language": map[string]any{
					"type":        "string",
					"description": "Which runtime interprets the source: python → python3, shell → bash.",
					"enum":        []string{"python", "shell"},
				},
				"code": map[string]any{
					"type":        "string",
					"description": "The complete source, run as a file: top-level statements execute in order and whatever it prints comes back. Credentials stay out of it — read them from the environment (os.environ[\"<KEY>\"] in python, \"$KEY\" in shell) and list the entries in secrets. Python gets the standard library only; an uninstalled import fails with ModuleNotFoundError rather than installing anything.",
				},
				"secrets": map[string]any{
					"type":        "array",
					"description": "Keychain entry names to expose as environment variables — ['POLYGON_API_KEY']. Each is injected under that exact name; beyond PATH and HOME nothing else from the host environment reaches the script, so a key the script reads but omits here is simply absent. A name missing from the keychain fails the call and names itself: store_secret it, then retry.",
					"items":       map[string]any{"type": "string"},
				},
				"stdin": map[string]any{
					"type":        "string",
					"description": "Text piped to the script's stdin. Omit when it takes no input.",
				},
			},
			"required": []string{"language", "code"},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Language string   `json:"language"`
				Code     string   `json:"code"`
				Secrets  []string `json:"secrets"`
				Stdin    string   `json:"stdin"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			return runScript(ctx, e, params.Language, params.Code, params.Secrets, params.Stdin)
		},
	})
}

func runScript(ctx context.Context, e *toolTypes.Executor, language, code string, secrets []string, stdin string) (string, error) {
	runtime, ok := scriptRuntimeMap[strings.TrimSpace(strings.ToLower(language))]
	if !ok {
		return "", fmt.Errorf("run_script must be python or shell, got %q", language)
	}
	if strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("run_script requires a non-empty 'code' string")
	}
	env := make([]string, 0, len(secrets)+3)
	for _, key := range secrets {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value := keychain.Get(key)
		if value == "" {
			return "", fmt.Errorf("%s is not in the keychain; call store_secret for it first", key)
		}
		env = append(env, key+"="+value)
	}

	dir, err := os.MkdirTemp(filesystem.StoreTempDir, "run_script-")
	if err != nil {
		return "", fmt.Errorf("os.MkdirTemp: %w", err)
	}
	defer os.RemoveAll(dir)

	scriptPath := filepath.Join(dir, "main"+runtime.suffix)
	if err := os.WriteFile(scriptPath, []byte(code), 0o600); err != nil {
		return "", fmt.Errorf("os.WriteFile: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, runScriptTimeout)
	defer cancel()

	cmd, err := go_pkg_sandbox.Wrap(ctx, runtime.binary, []string{scriptPath}, e.WorkDir, &go_pkg_sandbox.Option{
		Network: go_pkg_sandbox.NetworkAllow,
	})
	if err != nil {
		return "", fmt.Errorf("sandbox.Wrap: %w", err)
	}

	cmd.Env = append(env,
		"PATH="+os.Getenv("PATH"),
		"HOME="+os.Getenv("HOME"),
		"PYTHONUNBUFFERED=1",
	)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	sink := &progressWriter{send: toolTypes.Progress(ctx)}
	cmd.Stdout = sink
	cmd.Stderr = sink

	err = cmd.Run()
	output := sink.text()
	if err != nil {
		return fmt.Sprintf("%s\nError: %s", output, err.Error()), nil
	}
	if strings.TrimSpace(output) == "" {
		return "[exit 0] the script succeeded and printed nothing", nil
	}
	return output, nil
}
