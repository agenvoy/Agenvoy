package claudeCode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	provider "github.com/pardnchiu/go-llm-router/core"
	go_pkg_utils "github.com/pardnchiu/go-pkg/utils"
)

var usedPattern = regexp.MustCompile(`(?m)^Current (?:session|week \(all models\)): (\d+(?:\.\d+)?)% used`)

func Usage(ctx context.Context, _ provider.Config) (float64, error) {
	if err := CheckBinary(); err != nil {
		return 0, err
	}

	cmd := exec.CommandContext(ctx, "claude", "-p", "/usage",
		"--safe-mode", "--tools", "", "--strict-mcp-config", "--no-session-persistence")
	cmd.Dir = os.TempDir()
	stderr := &limitedBuffer{}
	cmd.Stderr = stderr
	raw, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("claude /usage: %w: %s", err, stderr.String())
	}

	text := string(raw)
	matches := usedPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("claude /usage: no subscription limits in output: %s", go_pkg_utils.TruncateString(strings.TrimSpace(text), 200))
	}
	used := 0.0
	for _, m := range matches {
		value, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, fmt.Errorf("strconv.ParseFloat %q: %w", m[1], err)
		}
		used = max(used, value)
	}
	return 100 - used, nil
}
