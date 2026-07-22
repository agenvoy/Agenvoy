package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/toolAdapter/mcp"
)

func ListMcpServers() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, err := mcp.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"servers": cfg.Servers})
	}
}

func McpStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		m := mcp.Manager()
		if m == nil {
			c.JSON(http.StatusOK, gin.H{"servers": []mcp.ServerInfo{}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"servers": m.Status("")})
	}
}

func McpHealth() gin.HandlerFunc {
	return func(c *gin.Context) {
		m := mcp.Manager()
		if m == nil {
			c.JSON(http.StatusOK, gin.H{"servers": []mcp.HealthInfo{}})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		c.JSON(http.StatusOK, gin.H{"servers": m.Health(ctx)})
	}
}

func McpReconnect() gin.HandlerFunc {
	return func(c *gin.Context) {
		m := mcp.Manager()
		if m == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no MCP manager"})
			return
		}
		if err := m.Reconnect(context.Background(), ""); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func SetMcpServer() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Name   string           `json:"name"`
			Server mcp.ServerConfig `json:"server"`
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
		if strings.TrimSpace(body.Server.Command) == "" && strings.TrimSpace(body.Server.URL) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "server.command or server.url is required"})
			return
		}

		cfg, err := mcp.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cfg.Servers[name] = body.Server
		if err := mcp.Save(cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func RemoveMcpServer() gin.HandlerFunc {
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

		cfg, err := mcp.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if _, ok := cfg.Servers[name]; !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
			return
		}
		delete(cfg.Servers, name)
		if err := mcp.Save(cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
