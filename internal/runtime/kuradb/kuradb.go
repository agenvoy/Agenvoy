package kuradb

import (
	"context"
	"debug/buildinfo"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const (
	BinaryPath        = "/usr/local/bin/kura"
	InstallURL        = "https://kuradb.agenvoy.com/scripts/install.sh"
	serviceKey        = "kuradb"
	healthInterval    = 1 * time.Minute
	healthRequestTime = 5 * time.Second
	healthMaxStrikes  = 3
)

type Runtime struct {
	UID       string `json:"uid"`
	PID       int    `json:"pid"`
	StartedAt string `json:"started_at"`
}

func Remove() error {
	if err := os.Remove(filesystem.KuradbEndpointPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("os.Remove %s: %w", filesystem.KuradbEndpointPath, err)
	}
	return nil
}

func SyncOpenAIKey(value string) error {
	if value == "" {
		return nil
	}
	switch runtime.GOOS {
	case "darwin":
		exec.Command("security", "delete-generic-password",
			"-s", serviceKey,
			"-a", "OPENAI_API_KEY").Run()
		cmd := exec.Command("security", "add-generic-password",
			"-s", serviceKey,
			"-a", "OPENAI_API_KEY",
			"-w", value)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("security add-generic-password: %s", strings.TrimSpace(string(out)))
		}
		return nil
	default:
		cmd := exec.Command("secret-tool", "store",
			"--label", serviceKey+"/OPENAI_API_KEY",
			"service", serviceKey, "account", "OPENAI_API_KEY")
		cmd.Stdin = strings.NewReader(value)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("secret-tool store: %s", strings.TrimSpace(string(out)))
		}
		return nil
	}
}

func IsInstalled() bool {
	info, err := os.Stat(BinaryPath)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular() && info.Mode()&0o111 != 0
}

var healthClient = &http.Client{
	Timeout: healthRequestTime,
}

func Health(ctx context.Context, onFail func()) {
	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()

	strikes := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := ProbeHealth(ctx); err != nil {
				strikes++
				slog.Warn("kuradb.RunHealth: probe failed",
					slog.Int("strike", strikes),
					slog.Int("max", healthMaxStrikes),
					slog.String("error", err.Error()))
				if strikes >= healthMaxStrikes {
					slog.Error("kuradb.RunHealth: 3 consecutive failures, disabling")
					if onFail != nil {
						onFail()
					}
					return
				}
				continue
			}
			if strikes > 0 {
				slog.Info("kuradb.RunHealth: recovered",
					slog.Int("prior_strikes", strikes))
			}
			strikes = 0
		}
	}
}

func ProbeHealth(ctx context.Context) error {
	if !IsRunning() {
		return fmt.Errorf("runtime.uid: process not alive")
	}

	base, err := filesystem.GetKuradbEndpoint()
	if err != nil {
		return fmt.Errorf("filesystem.GetKuradbEndpoint: %w", err)
	}
	reqCtx, cancel := context.WithTimeout(ctx, healthRequestTime)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, base+"/api/health", nil)
	if err != nil {
		return fmt.Errorf("http.NewRequestWithContext: %w", err)
	}
	resp, err := healthClient.Do(req)
	if err != nil {
		return fmt.Errorf("healthClient.Do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func Version() (string, error) {
	info, err := buildinfo.ReadFile(BinaryPath)
	if err != nil {
		return "", fmt.Errorf("buildinfo.ReadFile: %w", err)
	}
	return info.Main.Version, nil
}

func IsRunning() bool {
	if !go_pkg_filesystem_reader.Exists(filesystem.KuradbDir) {
		return false
	}
	r, err := go_pkg_filesystem.ReadJSON[Runtime](filesystem.KuradbUIDPath)
	if err != nil {
		return false
	}
	return isAlive(r.PID)
}

func isAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

const execTimeout = 15 * time.Second

func Start(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, BinaryPath)
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("kura: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func Stop(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, BinaryPath, "stop")
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("kura stop: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func Restart(ctx context.Context) error {
	if err := Stop(ctx); err != nil {
		return err
	}
	return Start(ctx)
}
