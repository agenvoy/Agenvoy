package tui

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/pardnchiu/agenvoy/internal/runtime/daemon"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

const daemonLogRetry = 3 * time.Second

var daemonLogPrefixes = []string{
	"Telegram Verification Code",
	"Discord Verification Code",
}

type daemonLogFrame struct {
	Type   string `json:"type"`
	Source string `json:"source"`
	Text   string `json:"text"`
}

func newDaemonLog(ctx context.Context) {
	client := &http.Client{}
	for {
		streamDaemonLog(ctx, client)
		select {
		case <-ctx.Done():
			return
		case <-time.After(daemonLogRetry):
		}
	}
}

func streamDaemonLog(ctx context.Context, client *http.Client) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, daemon.BaseURL()+"/v1/log?daemon=1&replay=0", nil)
	if err != nil {
		return utilsmsg.Wrap(utilsmsg.Internal, "http.NewRequestWithContext", "", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return utilsmsg.Wrap(utilsmsg.Unavailable, "client.Do", "", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return utilsmsg.New(utilsmsg.Upstream, "daemon log", "http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		data, ok := strings.CutPrefix(scanner.Text(), "data: ")
		if !ok {
			continue
		}
		var frame daemonLogFrame
		if err := json.Unmarshal([]byte(data), &frame); err != nil || frame.Type != "EventDaemonLog" {
			continue
		}
		if !slices.ContainsFunc(daemonLogPrefixes, func(prefix string) bool {
			return strings.HasPrefix(frame.Text, prefix)
		}) {
			continue
		}
		send(Log{
			Source: "daemon",
			Level:  frame.Source,
			Time:   time.Now(),
			Msg:    frame.Text,
		})
	}
	if err := scanner.Err(); err != nil {
		return utilsmsg.Wrap(utilsmsg.Unavailable, "scanner.Scan", "", err)
	}
	return nil
}
