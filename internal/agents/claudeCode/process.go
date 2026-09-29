package claudeCode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/configs"
)

const (
	idleTimeout   = 15 * time.Minute
	reapInterval  = time.Minute
	stderrLimit   = 8 << 10
	maxOutputLine = 64 << 20
)

var (
	poolMu    sync.Mutex
	pool      = map[string]*process{}
	reapStart sync.Once
)

type process struct {
	mu         sync.Mutex
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     *io.PipeReader
	lines      *bufio.Scanner
	stderr     *limitedBuffer
	exited     chan struct{}
	spec       string
	tools      string
	sent       []string
	lastAnswer string
	lastUse    time.Time
}

type limitedBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *limitedBuffer) Write(raw []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if room := stderrLimit - b.buf.Len(); room > 0 {
		b.buf.Write(raw[:min(len(raw), room)])
	}
	return len(raw), nil
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.TrimSpace(b.buf.String())
}

type resultLine struct {
	Type           string `json:"type"`
	Subtype        string `json:"subtype"`
	IsError        bool   `json:"is_error"`
	Result         string `json:"result"`
	APIErrorStatus int    `json:"api_error_status"`
	Usage          struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	} `json:"usage"`
}

var toolCallPattern = regexp.MustCompile(`(?s)<tool_call name="([^"]+)">(.*?)</tool_call>`)

func acquire(key string) *process {
	reapStart.Do(func() { go reap() })

	for {
		poolMu.Lock()
		p, ok := pool[key]
		if !ok {
			p = &process{}
			pool[key] = p
		}
		poolMu.Unlock()

		p.mu.Lock()
		poolMu.Lock()
		current := pool[key] == p
		poolMu.Unlock()
		if current {
			return p
		}
		p.mu.Unlock()
	}
}

func reap() {
	ticker := time.NewTicker(reapInterval)
	defer ticker.Stop()
	for range ticker.C {
		poolMu.Lock()
		for key, p := range pool {
			if !p.mu.TryLock() {
				continue
			}
			if !p.alive() || time.Since(p.lastUse) > idleTimeout {
				p.stop()
				delete(pool, key)
			}
			p.mu.Unlock()
		}
		poolMu.Unlock()
	}
}

func (p *process) start(model, effort string, withTools bool) error {
	args := []string{
		"-p",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--model", model,
		"--effort", effort,
		"--tools", "",
		"--strict-mcp-config",
		"--safe-mode",
		"--no-session-persistence",
	}
	if withTools {
		args = append(args, "--system-prompt", strings.TrimSpace(configs.ClaudeCodeToolPrompt))
	} else {
		args = append(args, "--system-prompt", strings.TrimSpace(configs.ClaudeCodePlainPrompt))
	}

	cmd := exec.Command("claude", args...)
	cmd.Dir = os.TempDir()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("cmd.StdinPipe: %w", err)
	}
	reader, writer := io.Pipe()
	cmd.Stdout = writer
	stderr := &limitedBuffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cmd.Start: %w", err)
	}

	exited := make(chan struct{})
	go func() {
		err := cmd.Wait()
		writer.CloseWithError(fmt.Errorf("claude exited: %v", err))
		close(exited)
	}()

	lines := bufio.NewScanner(reader)
	lines.Buffer(make([]byte, 64<<10), maxOutputLine)

	p.cmd, p.stdin, p.stdout, p.lines, p.stderr, p.exited = cmd, stdin, reader, lines, stderr, exited
	return nil
}

func (p *process) alive() bool {
	if p.cmd == nil {
		return false
	}
	select {
	case <-p.exited:
		return false
	default:
		return true
	}
}

func (p *process) stop() {
	if p.cmd == nil {
		return
	}
	_ = p.stdin.Close()
	_ = p.stdout.Close()
	_ = p.cmd.Process.Kill()
	<-p.exited
	p.cmd = nil
}

func (p *process) turn(ctx context.Context, content []map[string]any) (*provider.Output, int, error) {
	raw, err := json.Marshal(map[string]any{
		"type":    "user",
		"message": map[string]any{"role": "user", "content": content},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("json.Marshal: %w", err)
	}
	if _, err := p.stdin.Write(append(raw, '\n')); err != nil {
		return nil, 0, fmt.Errorf("write claude stdin: %w: %s", err, p.stderr.String())
	}

	done := make(chan struct{})
	var line *resultLine
	var readErr error
	go func() {
		defer close(done)
		line, readErr = p.readResult()
	}()

	select {
	case <-done:
	case <-ctx.Done():
		p.stop()
		<-done
		return nil, 0, ctx.Err()
	}

	if readErr != nil {
		return nil, 0, readErr
	}
	if line.IsError {
		return nil, line.APIErrorStatus, fmt.Errorf("claude error (%d): %s", line.APIErrorStatus, go_pkg_utils.TruncateString(line.Result, 1024))
	}
	return buildOutput(line)
}

func (p *process) readResult() (*resultLine, error) {
	for p.lines.Scan() {
		var line resultLine
		if json.Unmarshal(p.lines.Bytes(), &line) != nil || line.Type != "result" {
			continue
		}
		return &line, nil
	}
	if err := p.lines.Err(); err != nil {
		return nil, fmt.Errorf("read claude stdout: %w: %s", err, p.stderr.String())
	}
	return nil, fmt.Errorf("claude exited without a result: %s", p.stderr.String())
}

func buildOutput(line *resultLine) (*provider.Output, int, error) {
	message := provider.Message{
		Role:    "assistant",
		Content: strings.TrimSpace(toolCallPattern.ReplaceAllString(line.Result, "")),
	}
	for _, m := range toolCallPattern.FindAllStringSubmatch(line.Result, -1) {
		name := strings.TrimSpace(m[1])
		if name == "" {
			continue
		}
		args := strings.TrimSpace(m[2])
		if args == "" {
			args = "{}"
		}
		call := provider.ToolCall{ID: "call_" + strings.ReplaceAll(go_pkg_utils.UUID(), "-", "")[:24], Type: "function"}
		call.Function.Name = name
		call.Function.Arguments = args
		message.ToolCalls = append(message.ToolCalls, call)
	}

	finish := "stop"
	if len(message.ToolCalls) > 0 {
		finish = "tool_calls"
	} else if message.Content == "" {
		return nil, 0, fmt.Errorf("claude returned neither an answer nor tool calls")
	}

	u := line.Usage
	return &provider.Output{
		Choices: []provider.OutputChoices{{Message: message, FinishReason: finish}},
		Usage: provider.Usage{
			Input:       u.InputTokens,
			Output:      u.OutputTokens,
			CacheCreate: u.CacheCreationInputTokens,
			CacheRead:   u.CacheReadInputTokens,
		},
	}, 0, nil
}
