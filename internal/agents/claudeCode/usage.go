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

var usedPattern = regexp.MustCompile(`(?m)^Current (session|week \(all models\)): (\d+(?:\.\d+)?)% used`)

func Usage(ctx context.Context, _ provider.Config) (provider.UsageRemaining, error) {
	if err := CheckBinary(); err != nil {
		return provider.UsageRemaining{}, err
	}

	cmd := exec.CommandContext(ctx, "claude", "-p", "/usage",
		"--safe-mode", "--tools", "", "--strict-mcp-config", "--no-session-persistence")
	cmd.Dir = os.TempDir()
	stderr := &limitedBuffer{}
	cmd.Stderr = stderr
	raw, err := cmd.Output()
	if err != nil {
		return provider.UsageRemaining{}, fmt.Errorf("claude /usage: %w: %s", err, stderr.String())
	}

	text := string(raw)
	dic := map[string]float64{}
	for _, m := range usedPattern.FindAllStringSubmatch(text, -1) {
		value, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			return provider.UsageRemaining{}, fmt.Errorf("strconv.ParseFloat %q: %w", m[2], err)
		}
		dic[m[1]] = 100 - value
	}
	fiveHour, hasFiveHour := dic["session"]
	week, hasWeek := dic["week (all models)"]
	if !hasFiveHour || !hasWeek {
		return provider.UsageRemaining{}, fmt.Errorf("claude /usage: no subscription limits in output: %s", go_pkg_utils.TruncateString(strings.TrimSpace(text), 200))
	}
	return provider.UsageRemaining{FiveHour: &fiveHour, Week: &week}, nil
}
