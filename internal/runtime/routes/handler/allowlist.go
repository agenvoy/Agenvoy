package handler

import (
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/agents"
	allowSkill "github.com/pardnchiu/agenvoy/internal/agents/exec/allow/skill"
	allowTool "github.com/pardnchiu/agenvoy/internal/agents/exec/allow/tool"
	"github.com/pardnchiu/agenvoy/internal/runtime"
)

func ListAllowSkill() gin.HandlerFunc {
	return func(c *gin.Context) {
		scope := c.DefaultQuery("scope", "global")
		workDir := strings.TrimSpace(c.Query("work_dir"))

		scanner := agents.Scanner()
		if scanner == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "skill scanner unavailable"})
			return
		}
		names := scanner.List()
		sort.Strings(names)

		var allowed map[string]bool
		switch scope {
		case "global":
			allowed = allowSkill.LoadGlobal()
		case "project":
			if workDir == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "work_dir required for scope=project"})
				return
			}
			allowed = allowSkill.LoadEffective(workDir)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown scope: " + scope})
			return
		}

		source := make(map[string]string, len(names))
		for _, name := range names {
			if sk := scanner.Lookup(name); sk != nil {
				source[name] = runtime.SkillSource(sk.AbsPath)
			}
		}

		c.JSON(http.StatusOK, gin.H{"scope": scope, "skills": names, "allowed": allowed, "source": source})
	}
}

func ToggleAllowSkill() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Scope   string `json:"scope"`
			Name    string `json:"name"`
			WorkDir string `json:"work_dir"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
			return
		}

		var added bool
		var err error
		switch body.Scope {
		case "global", "":
			added, err = allowSkill.ToggleGlobal(name)
		case "project":
			workDir := strings.TrimSpace(body.WorkDir)
			if workDir == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "work_dir required for scope=project"})
				return
			}
			added, err = allowSkill.ToggleProject(workDir, name)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown scope: " + body.Scope})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "added": added})
	}
}

func ListAllowTool() gin.HandlerFunc {
	return func(c *gin.Context) {
		prefix := strings.TrimSpace(c.Query("prefix"))

		entries := make([]string, 0, 16)
		for entry := range allowTool.LoadGlobal() {
			if prefix == "" || strings.HasPrefix(entry, prefix) {
				entries = append(entries, entry)
			}
		}
		slices.Sort(entries)
		c.JSON(http.StatusOK, gin.H{"entries": entries})
	}
}

func SetAllowTool() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Prefix  string   `json:"prefix"`
			Entries []string `json:"entries"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		prefix := strings.TrimSpace(body.Prefix)
		if prefix == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "prefix is required"})
			return
		}

		entries := make([]string, 0, len(body.Entries))
		for _, entry := range body.Entries {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			if !strings.HasPrefix(entry, prefix) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "every entry must start with the prefix: " + entry})
				return
			}
			entries = append(entries, entry)
		}
		if slices.Contains(entries, prefix+"*") {
			entries = []string{prefix + "*"}
		}

		if err := allowTool.ReplaceGlobalPrefix(prefix, entries); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "entries": entries})
	}
}
