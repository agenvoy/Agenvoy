package handler

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"

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
			"files":       skillFiles(one.AbsPath),
			"deletable":   deletableSkill(one.AbsPath),
		})
	}
}

const skillFileMaxBytes = 256 << 10

var skillFileDirs = []string{"scripts", "references", "assets"}

type skillFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func skillFiles(skillPath string) []skillFile {
	dir := filepath.Dir(skillPath)
	out := make([]skillFile, 0)

	filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() || path == skillPath {
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Size() > skillFileMaxBytes {
			return nil
		}
		content, err := go_pkg_filesystem.ReadText(path)
		if err != nil || !utf8.ValidString(content) {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		root, _, ok := strings.Cut(rel, "/")
		if !ok || !slices.Contains(skillFileDirs, root) {
			return nil
		}
		out = append(out, skillFile{Path: rel, Content: content})
		return nil
	})

	slices.SortFunc(out, func(a, b skillFile) int {
		return strings.Compare(a.Path, b.Path)
	})
	return out
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
