package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
)

const mcpOAuthTimeout = 10 * time.Minute

func mcpOAuthServer(name string) (mcp.ServerConfig, error) {
	cfg, err := mcp.Load()
	if err != nil {
		return mcp.ServerConfig{}, err
	}
	server, ok := cfg.Servers[name]
	if !ok {
		return mcp.ServerConfig{}, fmt.Errorf("server %q not found", name)
	}
	if !server.Expand().IsHTTP() {
		return mcp.ServerConfig{}, fmt.Errorf("server %q is not an HTTP server; oauth applies to http/sse transports only", name)
	}
	return server, nil
}

func McpOAuthLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.TrimSpace(c.Query("name"))
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		if _, err := mcpOAuthServer(name); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		h := c.Writer.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no")
		c.Writer.WriteHeader(http.StatusOK)
		c.Writer.Flush()

		emit := func(v any) {
			raw, err := json.Marshal(v)
			if err != nil {
				return
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", raw)
			c.Writer.Flush()
		}

		urls := make(chan string, 1)
		done := make(chan error, 1)
		ctx, cancel := context.WithTimeout(c.Request.Context(), mcpOAuthTimeout)
		defer cancel()

		go func() {
			done <- mcp.Login(ctx, name, func(url string) {
				select {
				case urls <- url:
				default:
				}
			})
		}()

		for {
			select {
			case url := <-urls:
				emit(gin.H{"url": url})
			case err := <-done:
				if err != nil {
					emit(gin.H{"done": true, "ok": false, "error": err.Error()})
					return
				}
				result := gin.H{"done": true, "ok": true}
				reconnectCtx, reconnectCancel := context.WithTimeout(context.Background(), 30*time.Second)
				if rerr := mcp.Manager().ReconnectServer(reconnectCtx, name); rerr != nil {
					result["reconnect_error"] = rerr.Error()
				}
				reconnectCancel()
				emit(result)
				return
			}
		}
	}
}

func McpOAuthCallback() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		name := strings.TrimSpace(body.Name)
		url := strings.TrimSpace(body.URL)
		if name == "" || url == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name and url are required"})
			return
		}
		if err := mcp.SubmitCallback(name, url); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func McpOAuthClient() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Name         string `json:"name"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			RedirectURI  string `json:"redirect_uri"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		name := strings.TrimSpace(body.Name)
		clientID := strings.TrimSpace(body.ClientID)
		if name == "" || clientID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name and client_id are required"})
			return
		}
		if _, err := mcpOAuthServer(name); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		redirectURI := strings.TrimSpace(body.RedirectURI)
		if redirectURI == "" {
			redirectURI = mcp.DefaultRedirectURI
		}
		if err := mcp.ClearOAuth(name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := mcp.SaveOAuthClient(name, clientID, body.ClientSecret, redirectURI); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "redirect_uri": redirectURI})
	}
}

func McpOAuthClear() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		if err := mcp.ClearOAuth(name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
