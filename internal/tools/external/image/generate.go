package image

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

var mimeExt = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
}

type Request struct {
	Prompt      string
	AspectRatio string
	Size        string
	Quality     string
	Reference   string
	OutputFile  string
	WorkDir     string
}

func Generate(ctx context.Context, req Request) (string, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return "", fmt.Errorf("prompt is required")
	}

	img, agentName, err := agent(ctx)
	if err != nil {
		return "", err
	}

	opts := provider.ImageOptions{
		AspectRatio: strings.TrimSpace(req.AspectRatio),
		Size:        strings.TrimSpace(req.Size),
		Quality:     strings.TrimSpace(req.Quality),
	}
	if ref := strings.TrimSpace(req.Reference); ref != "" {
		b64, mime, err := readReference(ref)
		if err != nil {
			return "", err
		}
		opts.RefImageB64, opts.RefMime = b64, mime
	}

	result, err := img.GenerateImage(ctx, prompt, opts)
	if err != nil {
		return "", fmt.Errorf("%s GenerateImage: %w", agentName, err)
	}
	if result == nil || result.B64 == "" {
		return "", fmt.Errorf("%s returned no image data", agentName)
	}

	raw, err := base64.StdEncoding.DecodeString(result.B64)
	if err != nil {
		return "", fmt.Errorf("base64.Decode: %w", err)
	}

	path := outputPath(req, result.MimeType)
	if err := go_pkg_filesystem.CheckDir(filepath.Dir(path), true); err != nil {
		return "", fmt.Errorf("CheckDir: %w", err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		return "", fmt.Errorf("os.WriteFile [%s]: %w", path, err)
	}

	out := fmt.Sprintf("saved: %s\nmodel: %s\nbytes: %d", path, agentName, len(raw))
	if revised := strings.TrimSpace(result.Revised); revised != "" {
		out += "\nrevised prompt: " + revised
	}
	return out, nil
}

func outputPath(req Request, mime string) string {
	ext, ok := mimeExt[strings.ToLower(strings.TrimSpace(mime))]
	if !ok {
		ext = ".png"
	}

	target := strings.TrimSpace(req.OutputFile)
	switch {
	case target == "":
		target = "image-" + time.Now().Format("20060102-150405") + ext
	case filepath.Ext(target) == "":
		target += ext
	}
	if filepath.IsAbs(target) {
		return target
	}
	if dir := strings.TrimSpace(req.WorkDir); dir != "" {
		return filepath.Join(dir, target)
	}
	return filepath.Join(filesystem.DownloadDir, target)
}

func readReference(path string) (string, string, error) {
	if !filepath.IsAbs(path) {
		return "", "", fmt.Errorf("reference must be an absolute path")
	}
	if !go_pkg_filesystem_reader.Exists(path) {
		return "", "", fmt.Errorf("reference not found: %s", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("os.ReadFile [%s]: %w", path, err)
	}

	mime := "image/png"
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".webp":
		mime = "image/webp"
	}
	return base64.StdEncoding.EncodeToString(raw), mime, nil
}
