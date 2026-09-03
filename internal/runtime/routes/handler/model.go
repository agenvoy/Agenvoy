package handler

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	go_pkg_filesystem_reader "github.com/pardnchiu/go-pkg/filesystem/reader"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	configBot "github.com/pardnchiu/agenvoy/internal/session/config/bot"
	audioTool "github.com/pardnchiu/agenvoy/internal/tools/external/audio"
	imageTool "github.com/pardnchiu/agenvoy/internal/tools/external/image"
)

func modelNames() []string {
	names := []string{configBot.DefaultModel}
	if cfg, err := config.Load(); err == nil && cfg != nil {
		for _, m := range cfg.Models {
			if name := strings.TrimSpace(m.Name); name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}

func modelObject(name string) gin.H {
	owner := "agenvoy"
	if prefix, _, ok := strings.Cut(name, "@"); ok && prefix != "" {
		owner = prefix
	}
	return gin.H{
		"id":       name,
		"object":   "model",
		"created":  0,
		"owned_by": owner,
	}
}

func ListModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		models := modelNames()

		data := make([]gin.H, 0, len(models))
		for _, name := range models {
			data = append(data, modelObject(name))
		}

		c.JSON(http.StatusOK, gin.H{
			"models": models,
			"object": "list",
			"data":   data,
		})
	}
}

func GetModel() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimPrefix(c.Param("id"), "/")
		if id == "" || !slices.Contains(modelNames(), id) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"message": fmt.Sprintf("The model '%s' does not exist", id),
				"type":    "invalid_request_error",
				"param":   "model",
				"code":    "model_not_found",
			}})
			return
		}
		c.JSON(http.StatusOK, modelObject(id))
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

		firstModel := ""
		if len(cfg.Models) > 0 {
			firstModel = cfg.Models[0].Name
		}
		if cfg.DispatcherModel == name {
			cfg.DispatcherModel = firstModel
		}
		if cfg.SummaryModel == name {
			cfg.SummaryModel = firstModel
		}

		if err := config.Save(cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if dirs, err := go_pkg_filesystem_reader.ListDirs(filesystem.SessionsDir); err == nil {
			for _, dir := range dirs {
				sid := dir.Name
				if strings.HasPrefix(sid, ".") {
					continue
				}
				if model, _ := configBot.GetModel(sid); model == name {
					configBot.SetModel(sid, configBot.DefaultModel, "")
				}
			}
		}

		agents.Reload()

		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func modelRouting(c *gin.Context, cfg *config.Config) gin.H {
	return gin.H{
		"dispatcher":      cfg.DispatcherModel,
		"summary":         cfg.SummaryModel,
		"image":           cfg.ImageGenerator,
		"image_options":   imageTool.Available(c.Request.Context()),
		"image_providers": imageTool.Providers,
		"stt":             cfg.STTModel,
		"tts":             cfg.TTSModel,
		"audio_providers": audioTool.Providers,
	}
}

func ListAudioModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		c.JSON(http.StatusOK, gin.H{
			"stt_options": audioTool.STTOptions(ctx),
			"tts_options": audioTool.TTSOptions(ctx),
		})
	}
}

func GetModelRouting() gin.HandlerFunc {
	return func(c *gin.Context) {
		imageTool.Prune(c.Request.Context())

		cfg, err := config.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, modelRouting(c, cfg))
	}
}

func SetModelRouting() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Dispatcher *string `json:"dispatcher"`
			Summary    *string `json:"summary"`
			Image      *string `json:"image"`
			STT        *string `json:"stt"`
			TTS        *string `json:"tts"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cfg, err := config.Load()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		known := func(name string) bool {
			return slices.ContainsFunc(cfg.Models, func(m config.ModelEntry) bool { return m.Name == name })
		}

		dispatcher, summary := "", ""
		if body.Dispatcher != nil {
			dispatcher = strings.TrimSpace(*body.Dispatcher)
			if dispatcher != "" && !known(dispatcher) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unknown model: " + dispatcher})
				return
			}
		}
		if body.Summary != nil {
			summary = strings.TrimSpace(*body.Summary)
			if summary != "" && !known(summary) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unknown model: " + summary})
				return
			}
		}

		image := ""
		if body.Image != nil {
			image = strings.TrimSpace(*body.Image)
			if image == "off" {
				image = ""
			}
			if image != "" && !slices.Contains(imageTool.Available(c.Request.Context()), image) {
				if !slices.Contains(imageTool.Providers, image) {
					c.JSON(http.StatusBadRequest, gin.H{"error": "unknown image provider: " + image})
					return
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": image + " has no credentials stored"})
				return
			}
		}

		stt, tts := "", ""
		if body.STT != nil {
			stt = strings.TrimSpace(*body.STT)
			if stt != "" && !slices.Contains(audioTool.STTOptions(c.Request.Context()), stt) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unknown speech-to-text model: " + stt})
				return
			}
		}
		if body.TTS != nil {
			tts = strings.TrimSpace(*body.TTS)
			if tts != "" && !slices.Contains(audioTool.TTSOptions(c.Request.Context()), tts) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unknown text-to-speech model: " + tts})
				return
			}
		}

		if body.Dispatcher != nil {
			cfg.DispatcherModel = dispatcher
		}
		if body.Summary != nil {
			cfg.SummaryModel = summary
		}
		if body.Image != nil {
			cfg.ImageGenerator = image
		}
		if body.STT != nil {
			cfg.STTModel = stt
		}
		if body.TTS != nil {
			cfg.TTSModel = tts
		}
		if err := config.Save(cfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if body.Dispatcher != nil || body.Summary != nil {
			agents.Reload()
		}
		c.JSON(http.StatusOK, modelRouting(c, cfg))
	}
}
