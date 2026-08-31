package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const (
	daemonLogRetain = 28 * 24 * time.Hour
	daemonStdLayout = "2006/01/02 15:04:05"
	daemonArgLayout = "2006-01-02-15-04"
)

func GetDaemonLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := daemonRange(c.Query("from"), c.Query("to"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		path := filesystem.DaemonLogPath
		if !go_pkg_filesystem_reader.Exists(path) {
			c.JSON(http.StatusOK, gin.H{"content": ""})
			return
		}
		content, err := go_pkg_filesystem.ReadText(path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))
		c.JSON(http.StatusOK, gin.H{"content": recentDaemonLines(content, from, to, keyword)})
	}
}

func daemonRange(fromArg, toArg string) (time.Time, time.Time, error) {
	from := time.Now().Add(-daemonLogRetain)
	var to time.Time

	if arg := strings.TrimSpace(fromArg); arg != "" {
		at, err := time.ParseInLocation(daemonArgLayout, arg, time.Local)
		if err != nil {
			return from, to, fmt.Errorf("from must be yyyy-MM-dd-HH-mm")
		}
		from = at
	}
	if arg := strings.TrimSpace(toArg); arg != "" {
		at, err := time.ParseInLocation(daemonArgLayout, arg, time.Local)
		if err != nil {
			return from, to, fmt.Errorf("to must be yyyy-MM-dd-HH-mm")
		}
		to = at
	}
	return from, to, nil
}

func recentDaemonLines(content string, from, to time.Time, keyword string) string {
	var sb strings.Builder
	keep := false
	for line := range strings.SplitSeq(strings.TrimRight(content, "\n"), "\n") {
		if at, ok := daemonLineTime(line); ok {
			keep = !at.Before(from) &&
				(to.IsZero() || at.Before(to)) &&
				(keyword == "" || strings.Contains(strings.ToLower(line), keyword))
		}
		if !keep {
			continue
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func daemonLineTime(line string) (time.Time, bool) {
	if rest, ok := strings.CutPrefix(line, "time="); ok {
		field, _, _ := strings.Cut(rest, " ")
		at, err := time.Parse(time.RFC3339, field)
		return at, err == nil
	}
	if len(line) < len(daemonStdLayout) {
		return time.Time{}, false
	}
	at, err := time.ParseInLocation(daemonStdLayout, line[:len(daemonStdLayout)], time.Local)
	return at, err == nil
}
