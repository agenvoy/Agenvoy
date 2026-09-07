package webapp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/page"
)

type asset struct {
	Path string `json:"path"`
	URL  string `json:"url"`
}

func SyncAsset(ctx context.Context) error {
	stamp := filepath.Join(filesystem.VendorDir, ".version")
	if current, err := go_pkg_filesystem.ReadText(stamp); err == nil && current == runtime.CurrentVersion {
		return nil
	}

	raw, err := manifest()
	if err != nil {
		return err
	}

	var list []asset
	if err := json.Unmarshal(raw, &list); err != nil {
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	if err := go_pkg_filesystem.CheckDir(filesystem.VendorDir, true); err != nil {
		return fmt.Errorf("go_pkg_filesystem.CheckDir: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	for _, item := range list {
		body, _, err := go_pkg_http.GET[string](ctx, client, item.URL, nil)
		if err != nil {
			return fmt.Errorf("download %s: %w", item.Path, err)
		}
		if err := go_pkg_filesystem.WriteFile(filepath.Join(filesystem.VendorDir, item.Path), body, 0644); err != nil {
			return fmt.Errorf("write %s: %w", item.Path, err)
		}
		slog.Info("webapp asset downloaded",
			slog.String("path", item.Path),
			slog.Int("bytes", len(body)))
	}

	return go_pkg_filesystem.WriteFile(stamp, runtime.CurrentVersion, 0644)
}

func manifest() ([]byte, error) {
	if dir := os.Getenv("AGENVOY_PAGE_DIR"); dir != "" {
		content, err := go_pkg_filesystem.ReadText(filepath.Join(dir, "vendor.json"))
		if err != nil {
			return nil, fmt.Errorf("go_pkg_filesystem.ReadText: %w", err)
		}
		return []byte(content), nil
	}

	raw, err := page.FS.ReadFile("vendor.json")
	if err != nil {
		return nil, fmt.Errorf("page.FS.ReadFile: %w", err)
	}
	return raw, nil
}
