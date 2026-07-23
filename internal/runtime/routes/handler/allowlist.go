package handler

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/agents"
	allowCmd "github.com/pardnchiu/agenvoy/internal/agents/exec/allow/cmd"
	allowSkill "github.com/pardnchiu/agenvoy/internal/agents/exec/allow/skill"
)

func ListAllowCmd() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"white_list": allowCmd.List()})
	}
}

func AddAllowCmd() gin.HandlerFunc {
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
			return
		}
		added, err := allowCmd.Append(name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "added": added, "restart_required": added})
	}
}

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

		c.JSON(http.StatusOK, gin.H{"scope": scope, "skills": names, "allowed": allowed})
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
