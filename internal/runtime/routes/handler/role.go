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

func rolePath(name string) (string, error) {
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
	return filepath.Join(filesystem.PromptsDir, name+".md"), nil
}

type roleBody struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func ListRoles() gin.HandlerFunc {
	return func(c *gin.Context) {
		dir := filesystem.PromptsDir
		roles := make([]gin.H, 0)
		if !go_pkg_filesystem_reader.IsDir(dir) {
			// ! will deprecate in the future
			c.JSON(http.StatusOK, gin.H{"roles": roles, "rules": roles})
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
			role := gin.H{"name": strings.TrimSuffix(f.Name, ".md")}
			// * os.Stat retained: go-pkg exposes no accessor for size or mtime.
			if info, err := os.Stat(filepath.Join(dir, f.Name)); err == nil {
				role["size"] = info.Size()
				role["updated_at"] = info.ModTime().Unix()
			}
			roles = append(roles, role)
		}
		// ! will deprecate in the future
		c.JSON(http.StatusOK, gin.H{"roles": roles, "rules": roles})
	}
}

func GetRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		path, err := rolePath(strings.TrimPrefix(c.Param("name"), "/"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if !go_pkg_filesystem_reader.Exists(path) {
			c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
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

func CreateRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body roleBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		path, err := rolePath(body.Name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if go_pkg_filesystem_reader.Exists(path) {
			c.JSON(http.StatusConflict, gin.H{"error": "role already exists"})
			return
		}
		if err := go_pkg_filesystem.CheckDir(filesystem.PromptsDir, true); err != nil {
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

func UpdateRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			roleBody
			Rename string `json:"rename"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		path, err := rolePath(body.Name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if !go_pkg_filesystem_reader.Exists(path) {
			c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
			return
		}

		target := path
		if strings.TrimSpace(body.Rename) != "" {
			target, err = rolePath(body.Rename)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if target != path && go_pkg_filesystem_reader.Exists(target) {
				c.JSON(http.StatusConflict, gin.H{"error": "role already exists"})
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

func DeleteRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Query("name")
		if name == "" {
			var body roleBody
			if err := c.ShouldBindJSON(&body); err == nil {
				name = body.Name
			}
		}

		path, err := rolePath(name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if !go_pkg_filesystem_reader.Exists(path) {
			c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
			return
		}
		if err := go_pkg_filesystem.Remove(path); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}
