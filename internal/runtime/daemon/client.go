package daemon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const (
	requestTimeout = 10 * time.Second
	streamRetryMin = time.Second
	streamRetryMax = 30 * time.Second
	streamBufMax   = 4 * 1024 * 1024
)

var (
	client       = &http.Client{Timeout: requestTimeout}
	streamClient = &http.Client{}
	dataPrefix   = []byte("data: ")
)

var BaseURL = sync.OnceValue(func() string {
	return "http://127.0.0.1:" + filesystem.Port
})

func Get[T any](ctx context.Context, path string, query url.Values) (T, error) {
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	out, _, err := go_pkg_http.GET[T](ctx, client, BaseURL()+path, nil)
	return out, err
}

func Post[T any](ctx context.Context, path string, body map[string]any) (T, error) {
	out, _, err := go_pkg_http.POST[T](ctx, client, BaseURL()+path, nil, body, "")
	return out, err
}

func Delete[T any](ctx context.Context, path string, body map[string]any) (T, error) {
	out, _, err := go_pkg_http.DELETE[T](ctx, client, BaseURL()+path, nil, body, "")
	return out, err
}

func Stream(ctx context.Context, path string, onData func([]byte)) {
	backoff := streamRetryMin

	for ctx.Err() == nil {
		resp, err := go_pkg_http.GETStream(ctx, streamClient, BaseURL()+path, map[string]string{"Accept": "text/event-stream"})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			waitBackoff(ctx, &backoff)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			waitBackoff(ctx, &backoff)
			continue
		}
		backoff = streamRetryMin

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 64*1024), streamBufMax)
		for scanner.Scan() {
			if line := scanner.Bytes(); bytes.HasPrefix(line, dataPrefix) {
				onData(line[len(dataPrefix):])
			}
		}
		if err := scanner.Err(); err != nil && ctx.Err() == nil {
			slog.Debug("daemon stream",
				slog.String("path", path),
				slog.String("error", err.Error()))
		}
		resp.Body.Close()
	}
}

func waitBackoff(ctx context.Context, backoff *time.Duration) {
	select {
	case <-time.After(*backoff):
	case <-ctx.Done():
	}
	*backoff = min(*backoff*2, streamRetryMax)
}

func Publish(ctx context.Context, path string, body any) {
	raw, err := json.Marshal(body)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BaseURL()+path, bytes.NewReader(raw))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
