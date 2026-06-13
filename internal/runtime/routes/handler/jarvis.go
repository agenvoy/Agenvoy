package handler

import (
	_ "embed"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"

	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

const jarvisSessionID = "jarvis"

//go:embed jarvis.html
var jarvisShellHTML string

const defaultPageHTML = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><title>agenvoy</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{min-height:100vh;display:flex;align-items:center;justify-content:center;background:#06090d;font-family:Inter,-apple-system,sans-serif}
.greeting{text-align:center;color:#e2e8f0}
.greeting h1{font-size:28px;font-weight:600;margin-bottom:8px}
.greeting p{font-size:15px;color:#64748b}
</style>
</head>
<body>
<div class="greeting">
<h1>Hi, I'm Jarvis</h1>
<p>Your personal agent — type below to start.</p>
</div>
</body>
</html>`

func Jarvis() gin.HandlerFunc {
	return func(c *gin.Context) {
		pageDir := filesystem.PagePath(jarvisSessionID)
		_ = go_pkg_filesystem.CheckDir(pageDir, true)

		indexPath := filepath.Join(pageDir, "index.html")
		if _, err := go_pkg_filesystem.ReadText(indexPath); err != nil {
			_ = go_pkg_filesystem.WriteFile(indexPath, defaultPageHTML, 0644)
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(jarvisShellHTML))
	}
}

func JarvisReset() gin.HandlerFunc {
	return func(c *gin.Context) {
		indexPath := filepath.Join(filesystem.PagePath(jarvisSessionID), "index.html")
		if err := go_pkg_filesystem.WriteFile(indexPath, defaultPageHTML, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func JarvisListener() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()

		sub := pubsub.Sub(jarvisSessionID, 16)
		defer sub.Close()

		ctx := c.Request.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sub.Events():
				if !ok {
					return
				}
				if ev.Type == agentTypes.EventTextDone || ev.Type == agentTypes.EventDone {
					fmt.Fprintf(c.Writer, "data: {\"type\":%q}\n\n", ev.Type)
					c.Writer.Flush()
				}
			}
		}
	}
}

func JarvisPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		fp := c.Param("filepath")
		if fp == "" || fp == "/" {
			fp = "/index.html"
		}

		pageDir := filesystem.PagePath(jarvisSessionID)
		absPath := filepath.Join(pageDir, filepath.Clean(fp))

		if !go_pkg_filesystem_reader.Exists(absPath) {
			c.Status(http.StatusNotFound)
			return
		}

		c.File(absPath)
	}
}
