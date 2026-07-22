package handler

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
)

func ListModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		models := []string{configBot.DefaultModel}
		if cfg, err := config.Load(); err == nil && cfg != nil {
			for _, m := range cfg.Models {
				if name := strings.TrimSpace(m.Name); name != "" {
					models = append(models, name)
				}
			}
		}
		c.JSON(http.StatusOK, gin.H{"models": models})
	}
}

func validModelPrefix(prefix string) bool {
	if prefix == "" {
		return false
	}
	if strings.HasPrefix(prefix, "compat[") && strings.HasSuffix(prefix, "]") {
		return len(prefix) > len("compat[]")
	}
	return findProvider(prefix) != nil
}

func AddModel() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Prefix string   `json:"prefix"`
			Models []string `json:"models"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		prefix := strings.TrimSpace(body.Prefix)
		if !validModelPrefix(prefix) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "prefix must be a known provider id or oauth alias (see GET /v1/providers)"})
			return
		}

		selected := make(map[string]bool, len(body.Models))
		for _, id := range body.Models {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			fullName := prefix + "@" + id
			if fullName == configBot.DefaultModel {
				c.JSON(http.StatusBadRequest, gin.H{"error": "model cannot be the reserved default \"auto\""})
				return
			}
			selected[fullName] = true
		}

		cfg, err := config.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		fullPrefix := prefix + "@"
		var kept []config.ModelEntry
		var added, removed []string
		for _, m := range cfg.Models {
			if strings.HasPrefix(m.Name, fullPrefix) {
				if selected[m.Name] {
					kept = append(kept, m)
					delete(selected, m.Name)
				} else {
					removed = append(removed, m.Name)
				}
			} else {
				kept = append(kept, m)
			}
		}
		for fullName := range selected {
			kept = append(kept, config.ModelEntry{Name: fullName})
			added = append(added, fullName)
		}
		slices.Sort(added)
		slices.Sort(removed)

		cfg.Models = kept
		if cfg.DispatcherModel != "" && strings.HasPrefix(cfg.DispatcherModel, fullPrefix) {
			found := slices.ContainsFunc(kept, func(m config.ModelEntry) bool { return m.Name == cfg.DispatcherModel })
			if !found {
				cfg.DispatcherModel = ""
			}
		}
		if cfg.DispatcherModel == "" && len(kept) > 0 {
			cfg.DispatcherModel = kept[0].Name
		}

		if err := config.Save(cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		agents.Reload()

		c.JSON(http.StatusOK, gin.H{"ok": true, "added": added, "removed": removed})
	}
}

func RemoveModel() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.TrimPrefix(c.Param("name"), "/")
		name = strings.TrimSpace(name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		cfg, err := config.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		idx := slices.IndexFunc(cfg.Models, func(m config.ModelEntry) bool { return m.Name == name })
		if idx == -1 {
			c.JSON(http.StatusNotFound, gin.H{"error": "model not found"})
			return
		}
		cfg.Models = slices.Delete(cfg.Models, idx, idx+1)

		if cfg.DispatcherModel == name {
			cfg.DispatcherModel = ""
			if len(cfg.Models) > 0 {
				cfg.DispatcherModel = cfg.Models[0].Name
			}
		}
		if cfg.SummaryModel == name {
			cfg.SummaryModel = ""
		}

		if err := config.Save(cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		agents.Reload()

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func SetSessionModel() gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := strings.TrimSpace(c.Param("session_id"))
		if sid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}
		if !go_pkg_filesystem_reader.Exists(filesystem.SessionDir(sid)) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}

		var body struct {
			Model string `json:"model"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		model := strings.TrimSpace(body.Model)
		if model == "" {
			model = configBot.DefaultModel
		}
		if model != configBot.DefaultModel {
			valid := false
			if cfg, err := config.Load(); err == nil && cfg != nil {
				for _, m := range cfg.Models {
					if m.Name == model {
						valid = true
						break
					}
				}
			}
			if !valid {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unknown model: " + model})
				return
			}
		}

		configBot.SetModel(sid, model, "")
		c.JSON(http.StatusOK, gin.H{"ok": true, "model": model})
	}
}
