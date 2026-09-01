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

func allowSkillBlock(c *gin.Context) (gin.H, bool) {
	scope := c.DefaultQuery("scope", "global")
	workDir := strings.TrimSpace(c.Query("work_dir"))

	scanner := agents.Scanner()
	if scanner == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "skill scanner unavailable"})
		return nil, false
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
			return nil, false
		}
		allowed = allowSkill.LoadEffective(workDir)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown scope: " + scope})
		return nil, false
	}

	source := make(map[string]string, len(names))
	for _, name := range names {
		if sk := scanner.Lookup(name); sk != nil {
			source[name] = runtime.SkillSource(sk.AbsPath)
		}
	}
	return gin.H{"scope": scope, "skills": names, "allowed": allowed, "source": source}, true
}

func allowToolBlock(c *gin.Context) gin.H {
	prefix := strings.TrimSpace(c.Query("prefix"))

	entries := make([]string, 0, 16)
	for entry := range allowTool.LoadGlobal() {
		if prefix == "" || strings.HasPrefix(entry, prefix) {
			entries = append(entries, entry)
		}
	}
	slices.Sort(entries)
	return gin.H{"entries": entries}
}

func GetAllowlist() gin.HandlerFunc {
	return func(c *gin.Context) {
		skill, ok := allowSkillBlock(c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{"skill": skill, "tool": allowToolBlock(c)})
	}
}

func SetAllowlist() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Skill *struct {
				Name    string `json:"name"`
				Scope   string `json:"scope"`
				WorkDir string `json:"work_dir"`
			} `json:"skill"`
			Tool *struct {
				Prefix  string   `json:"prefix"`
				Entries []string `json:"entries"`
			} `json:"tool"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if body.Skill == nil && body.Tool == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "skill or tool is required"})
			return
		}

		out := gin.H{"ok": true}

		if body.Tool != nil {
			prefix := strings.TrimSpace(body.Tool.Prefix)
			if prefix == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "tool.prefix is required"})
				return
			}
			entries := make([]string, 0, len(body.Tool.Entries))
			for _, entry := range body.Tool.Entries {
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
			out["tool"] = gin.H{"entries": entries}
		}

		if body.Skill != nil {
			name := strings.TrimSpace(body.Skill.Name)
			if name == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "skill.name is required"})
				return
			}
			var added bool
			var err error
			switch body.Skill.Scope {
			case "global", "":
				added, err = allowSkill.ToggleGlobal(name)
			case "project":
				workDir := strings.TrimSpace(body.Skill.WorkDir)
				if workDir == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "skill.work_dir required for scope=project"})
					return
				}
				added, err = allowSkill.ToggleProject(workDir, name)
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": "unknown scope: " + body.Skill.Scope})
				return
			}
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out["skill"] = gin.H{"added": added}
		}

		c.JSON(http.StatusOK, out)
	}
}
