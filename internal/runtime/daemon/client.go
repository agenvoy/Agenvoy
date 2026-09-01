package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"

	go_pkg_http "github.com/pardnchiu/go-pkg/http"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const (
	requestTimeout = 10 * time.Second
)

var (
	client = &http.Client{Timeout: requestTimeout}
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
