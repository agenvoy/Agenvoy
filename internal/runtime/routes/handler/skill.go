package handler

import (
	"net/http"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/internal/tools"
)

func ListSkills() gin.HandlerFunc {
	return func(c *gin.Context) {
		list := make([]gin.H, 0)

		scanner := agents.Scanner()
		if scanner == nil {
			c.JSON(http.StatusOK, gin.H{"skills": list})
			return
		}
		scanner.Scan()

		for _, name := range scanner.List() {
			if name == "" || slices.Contains(tools.TUIOnlySkills, name) {
				continue
			}
			item := gin.H{"name": name}
			if s := scanner.Lookup(name); s != nil && s.Description != "" {
				item["description"] = s.Description
			}
			list = append(list, item)
		}
		c.JSON(http.StatusOK, gin.H{"skills": list})
	}
}

func GetSkill() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.Trim(c.Param("name"), "/")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		scanner := agents.Scanner()
		if scanner == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "skill scanner unavailable"})
			return
		}
		scanner.Scan()

		one := scanner.Lookup(name)
		if one == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "skill not found: " + name})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"name":        one.Name,
			"description": one.Description,
			"path":        one.AbsPath,
			"source":      runtime.SkillSource(one.AbsPath),
			"content":     one.Content,
			"deletable":   deletableSkill(one.AbsPath),
		})
	}
}

func DeleteSkill() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.TrimSpace(c.Query("name"))
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		scanner := agents.Scanner()
		if scanner == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "skill scanner unavailable"})
			return
		}
		scanner.Scan()

		one := scanner.Lookup(name)
		if one == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "skill not found: " + name})
			return
		}
		if !deletableSkill(one.AbsPath) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only skills under " + filesystem.SkillsDir + " can be deleted"})
			return
		}

		dir := filepath.Dir(one.AbsPath)
		trashPath, err := filesystem.TrashDir(dir, filesystem.SkillTrashDir, filepath.Base(dir))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		scanner.Scan()

		c.JSON(http.StatusOK, gin.H{"ok": true, "trashed": trashPath})
	}
}

func deletableSkill(absPath string) bool {
	if absPath == "" || filesystem.SkillsDir == "" {
		return false
	}
	if filesystem.SystemSkillsDir != "" && strings.HasPrefix(absPath, filesystem.SystemSkillsDir+"/") {
		return false
	}
	return strings.HasPrefix(absPath, filesystem.SkillsDir+"/")
}
