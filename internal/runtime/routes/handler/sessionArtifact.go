package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	usagelog "github.com/pardnchiu/agenvoy/internal/session/usage"
)

var usagePeriods = []struct {
	label string
	days  int
}{
	{label: "24h", days: 1},
	{label: "7d", days: 7},
	{label: "28d", days: 28},
}

func GetSessionDaemonLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		if sid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}
		if !go_pkg_filesystem_reader.Exists(filesystem.DaemonLogPath) {
			c.JSON(http.StatusOK, gin.H{"lines": []string{}})
			return
		}
		content, err := go_pkg_filesystem.ReadText(filesystem.DaemonLogPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		limit, err := strconv.Atoi(c.DefaultQuery("limit", "200"))
		if err != nil || limit <= 0 {
			limit = 200
		}

		var matched []string
		for line := range strings.SplitSeq(content, "\n") {
			if strings.Contains(line, sid) {
				matched = append(matched, line)
			}
		}
		if len(matched) > limit {
			matched = matched[len(matched)-limit:]
		}
		c.JSON(http.StatusOK, gin.H{"lines": matched})
	}
}

func GetSessionActionLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		if sid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}
		path := filesystem.ActionLogPath(sid)
		if !go_pkg_filesystem_reader.Exists(path) {
			c.JSON(http.StatusOK, gin.H{"content": ""})
			return
		}
		content, err := go_pkg_filesystem.ReadText(path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"content": content})
	}
}

func GetSessionUsageLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		if sid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}
		path := filesystem.UsageLogPath(sid)
		now := time.Now()

		periods := make(map[string]map[string]usagelog.ModelUsage, len(usagePeriods))
		for _, period := range usagePeriods {
			summary, err := usagelog.Usage(path, period.days, now)
			if err != nil {
				if os.IsNotExist(err) {
					summary = map[string]usagelog.ModelUsage{}
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
			periods[period.label] = summary
		}
		c.JSON(http.StatusOK, gin.H{"periods": periods})
	}
}

func ListSessionHistoryFiles() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		if sid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}
		dir := filesystem.TaskHistoryDir(sid)
		if !go_pkg_filesystem_reader.Exists(dir) {
			c.JSON(http.StatusOK, gin.H{"files": []go_pkg_filesystem_reader.File{}})
			return
		}
		files, err := go_pkg_filesystem_reader.ListFiles(dir)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"files": files})
	}
}

func GetSessionHistoryFile() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		name := strings.TrimPrefix(c.Param("file"), "/")
		if sid == "" || name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and file are required"})
			return
		}
		if name != filepath.Base(name) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file name"})
			return
		}

		path := filepath.Join(filesystem.TaskHistoryDir(sid), name)
		if !go_pkg_filesystem_reader.Exists(path) {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
			return
		}
		content, err := go_pkg_filesystem.ReadText(path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"content": content})
	}
}
