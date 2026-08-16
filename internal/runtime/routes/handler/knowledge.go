package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem "github.com/pardnchiu/go-pkg/filesystem"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

func knowledgePath(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".md")

	switch {
	case name == "":
		return "", fmt.Errorf("name is required")
	case strings.ContainsAny(name, `/\`):
		return "", fmt.Errorf("name cannot contain a path separator")
	case name == "." || name == "..":
		return "", fmt.Errorf("name cannot be a path segment")
	case strings.HasPrefix(name, "."):
		return "", fmt.Errorf("name cannot start with a dot")
	}
	return filepath.Join(filesystem.KnowledgeDir, name+".md"), nil
}

type knowledgeBody struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func ListKnowledges() gin.HandlerFunc {
	return func(c *gin.Context) {
		dir := filesystem.KnowledgeDir
		list := make([]gin.H, 0)
		if !go_pkg_filesystem_reader.IsDir(dir) {
			c.JSON(http.StatusOK, gin.H{"knowledges": list})
			return
		}

		files, err := go_pkg_filesystem_reader.ListFiles(dir)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, f := range files {
			if !strings.HasSuffix(f.Name, ".md") || strings.HasPrefix(f.Name, ".") {
				continue
			}
			path := filepath.Join(dir, f.Name)
			entry := gin.H{"name": strings.TrimSuffix(f.Name, ".md")}
			if info, err := os.Stat(path); err == nil {
				entry["size"] = info.Size()
				entry["updated_at"] = info.ModTime().Unix()
			}
			list = append(list, entry)
		}
		c.JSON(http.StatusOK, gin.H{"knowledges": list})
	}
}

func GetKnowledge() gin.HandlerFunc {
	return func(c *gin.Context) {
		path, err := knowledgePath(strings.TrimPrefix(c.Param("name"), "/"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if !go_pkg_filesystem_reader.Exists(path) {
			c.JSON(http.StatusNotFound, gin.H{"error": "knowledge not found"})
			return
		}

		content, err := go_pkg_filesystem.ReadText(path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"name":    strings.TrimSuffix(filepath.Base(path), ".md"),
			"content": content,
		})
	}
}

func CreateKnowledge() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body knowledgeBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		path, err := knowledgePath(body.Name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if go_pkg_filesystem_reader.Exists(path) {
			c.JSON(http.StatusConflict, gin.H{"error": "knowledge already exists"})
			return
		}
		if err := go_pkg_filesystem.CheckDir(filesystem.KnowledgeDir, true); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := go_pkg_filesystem.WriteFile(path, body.Content, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": strings.TrimSuffix(filepath.Base(path), ".md")})
	}
}

func UpdateKnowledge() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			knowledgeBody
			Rename string `json:"rename"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		path, err := knowledgePath(body.Name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if !go_pkg_filesystem_reader.Exists(path) {
			c.JSON(http.StatusNotFound, gin.H{"error": "knowledge not found"})
			return
		}

		target := path
		if strings.TrimSpace(body.Rename) != "" {
			target, err = knowledgePath(body.Rename)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if target != path && go_pkg_filesystem_reader.Exists(target) {
				c.JSON(http.StatusConflict, gin.H{"error": "knowledge already exists"})
				return
			}
		}

		if err := go_pkg_filesystem.WriteFile(target, body.Content, 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if target != path {
			if err := go_pkg_filesystem.Remove(path); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"name": strings.TrimSuffix(filepath.Base(target), ".md")})
	}
}

func DeleteKnowledge() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Query("name")
		if name == "" {
			var body knowledgeBody
			if err := c.ShouldBindJSON(&body); err == nil {
				name = body.Name
			}
		}

		path, err := knowledgePath(name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if !go_pkg_filesystem_reader.Exists(path) {
			c.JSON(http.StatusNotFound, gin.H{"error": "knowledge not found"})
			return
		}
		if err := go_pkg_filesystem.Remove(path); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
