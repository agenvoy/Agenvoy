package mcp

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

type ServerConfig struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type Config struct {
	Servers map[string]ServerConfig `json:"servers,omitempty"`
}

func Load() (Config, error) {
	if !go_pkg_filesystem_reader.Exists(filesystem.McpPath) {
		return Config{Servers: map[string]ServerConfig{}}, nil
	}
	cfg, err := go_pkg_filesystem.ReadJSON[Config](filesystem.McpPath)
	if err != nil {
		return Config{}, fmt.Errorf("go_pkg_filesystem.ReadJSON: %w", err)
	}
	if cfg.Servers == nil {
		cfg.Servers = map[string]ServerConfig{}
	}
	return cfg, nil
}

func Save(cfg Config) error {
	if cfg.Servers == nil {
		cfg.Servers = map[string]ServerConfig{}
	}
	if err := go_pkg_filesystem.CheckDir(filepath.Dir(filesystem.McpPath), true); err != nil {
		return fmt.Errorf("go_pkg_filesystem.CheckDir: %w", err)
	}
	if err := go_pkg_filesystem.WriteJSON(filesystem.McpPath, cfg, true); err != nil {
		return fmt.Errorf("go_pkg_filesystem.WriteJSON: %w", err)
	}
	return nil
}

func (c ServerConfig) Expand() ServerConfig {
	out := ServerConfig{
		Command: c.Command,
		URL:     c.URL,
	}
	if len(c.Args) > 0 {
		out.Args = make([]string, len(c.Args))
		for i, a := range c.Args {
			out.Args[i] = os.ExpandEnv(a)
		}
	}
	if len(c.Env) > 0 {
		out.Env = make(map[string]string, len(c.Env))
		for k, v := range c.Env {
			out.Env[k] = os.ExpandEnv(v)
		}
	}
	if len(c.Headers) > 0 {
		out.Headers = make(map[string]string, len(c.Headers))
		for k, v := range c.Headers {
			out.Headers[k] = normalizeHeaderValue(k, os.ExpandEnv(v))
		}
	}
	return out
}

func normalizeHeaderValue(key, value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !strings.EqualFold(strings.TrimSpace(key), "Authorization") {
		return value
	}
	if len(strings.Fields(value)) > 1 {
		return value
	}
	return "Bearer " + value
}

func (c ServerConfig) IsHTTP() bool {
	return strings.TrimSpace(c.URL) != ""
}

func (c ServerConfig) IsStdio() bool {
	return strings.TrimSpace(c.Command) != ""
}

func (c ServerConfig) toTransport() (mcpsdk.Transport, error) {
	switch {
	case c.IsHTTP():
		return &mcpsdk.StreamableClientTransport{
			Endpoint:   strings.TrimSpace(c.URL),
			HTTPClient: headerClient(c.Headers),
		}, nil
	case c.IsStdio():
		cmd := exec.Command(c.Command, c.Args...)
		cmd.Env = os.Environ()
		for key, value := range c.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Stderr = os.Stderr
		return &mcpsdk.CommandTransport{Command: cmd}, nil
	default:
		return nil, fmt.Errorf("neither command nor url")
	}
}

type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (r *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for key, value := range r.headers {
		req.Header.Set(key, value)
	}
	return r.base.RoundTrip(req)
}

func headerClient(headers map[string]string) *http.Client {
	transport := &http.Transport{}
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = base.Clone()
	}
	transport.ResponseHeaderTimeout = 60 * time.Second

	var roundTripper http.RoundTripper = transport
	if len(headers) > 0 {
		roundTripper = &headerRoundTripper{base: transport, headers: headers}
	}
	return &http.Client{Transport: roundTripper}
}
