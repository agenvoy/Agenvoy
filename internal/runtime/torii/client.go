package torii

import (
	"context"
	"log/slog"
	"net/url"
	"strconv"

	"github.com/pardnchiu/agenvoy/internal/runtime/daemon"
)

type Entry struct {
	Key string `json:"key"`
	Val string `json:"value"`
}

func (e *Entry) Value() string {
	return e.Val
}

type ScanOption struct {
	Contains string
	After    int64
	Limit    int
}

type Client struct {
	db int
}

func Remote(idx int) *Client {
	return &Client{db: idx}
}

func (c *Client) path(key string) string {
	return "/v1/toriidb/" + strconv.Itoa(c.db) + "/" + url.PathEscape(key)
}

func (c *Client) Get(ctx context.Context, key string) (*Entry, bool) {
	body, err := daemon.Get[struct {
		Entry *Entry `json:"entry"`
	}](ctx, c.path(key), nil)
	if err != nil || body.Entry == nil {
		return nil, false
	}
	return body.Entry, true
}

func (c *Client) Keys(ctx context.Context, pattern string) []string {
	body, err := daemon.Get[struct {
		Keys []string `json:"keys"`
	}](ctx, c.path(pattern), url.Values{"keys": {"1"}})
	if err != nil {
		slog.Debug("torii.Client.Keys", slog.String("error", err.Error()))
		return nil
	}
	return body.Keys
}

func (c *Client) Scan(ctx context.Context, pattern string, opt ScanOption) []Entry {
	query := url.Values{}
	if opt.Contains != "" {
		query.Set("contains", opt.Contains)
	}
	if opt.After > 0 {
		query.Set("after", strconv.FormatInt(opt.After, 10))
	}
	if opt.Limit > 0 {
		query.Set("limit", strconv.Itoa(opt.Limit))
	}

	body, err := daemon.Get[struct {
		Entries []Entry `json:"entries"`
	}](ctx, c.path(pattern), query)
	if err != nil {
		slog.Debug("torii.Client.Scan", slog.String("error", err.Error()))
		return nil
	}
	return body.Entries
}

func (c *Client) VSearch(ctx context.Context, text, pattern string, k int) ([]string, error) {
	body, err := daemon.Get[struct {
		Keys []string `json:"keys"`
	}](ctx, c.path(pattern), url.Values{"search": {text}, "limit": {strconv.Itoa(k)}})
	if err != nil {
		return nil, err
	}
	return body.Keys, nil
}

func (c *Client) Set(ctx context.Context, key, value string, expireAt *int64) error {
	return c.write(ctx, key, value, expireAt, false)
}

func (c *Client) SetVector(ctx context.Context, key, value string, expireAt *int64) error {
	return c.write(ctx, key, value, expireAt, true)
}

func (c *Client) Expire(ctx context.Context, key string, seconds int64) error {
	return c.ExpireMany(ctx, []string{key}, seconds)
}

func (c *Client) ExpireMany(ctx context.Context, keys []string, seconds int64) error {
	if len(keys) == 0 {
		return nil
	}
	_, err := daemon.Post[struct{}](ctx, "/v1/toriidb", map[string]any{
		"db":   c.db,
		"keys": keys,
		"ttl":  seconds,
	})
	return err
}

func (c *Client) write(ctx context.Context, key, value string, expireAt *int64, vector bool) error {
	_, err := daemon.Post[struct{}](ctx, "/v1/toriidb", map[string]any{
		"db":        c.db,
		"key":       key,
		"value":     value,
		"expire_at": expireAt,
		"vector":    vector,
	})
	return err
}

func (c *Client) Del(ctx context.Context, keys ...string) int {
	body, err := daemon.Delete[struct {
		Removed int `json:"removed"`
	}](ctx, "/v1/toriidb", map[string]any{"db": c.db, "keys": keys})
	if err != nil {
		slog.Debug("torii.Client.Del", slog.String("error", err.Error()))
		return 0
	}
	return body.Removed
}
