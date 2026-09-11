package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/startup"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

const maxReplyLang = 64

func GetConfig() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Param("target") {
		case "startup":
			c.JSON(http.StatusOK, gin.H{"enabled": startup.State(), "installed": startup.Enabled()})
		case "system":
			c.JSON(http.StatusOK, gin.H{
				"reply_lang": filesystem.ConfigReplyLang,
				"languages":  filesystem.ReplyLangOptions(),
			})
		case "output_dir":
			c.JSON(http.StatusOK, gin.H{
				"output_dir": filesystem.ConfigOutputDir,
				"resolved":   filesystem.OutputDir(),
			})
		default:
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown config target"})
		}
	}
}

func SetConfig() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Param("target") {
		case "startup":
			setStartup(c)
		case "system":
			setSystemConfig(c)
		case "output_dir":
			setOutputDir(c)
		default:
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown config target"})
		}
	}
}

func setStartup(c *gin.Context) {
	var body struct {
		Enable *bool `json:"enable"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Enable == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "enable is required"})
		return
	}

	var (
		detail string
		err    error
	)
	if *body.Enable {
		detail, err = startup.Enable()
	} else {
		detail, err = startup.Disable()
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "enabled": startup.State(), "installed": startup.Enabled(), "detail": detail})
}

func setSystemConfig(c *gin.Context) {
	var body struct {
		ReplyLang *string `json:"reply_lang"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.ReplyLang == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reply_lang is required"})
		return
	}

	lang := filesystem.CanonicalReplyLang(*body.ReplyLang)
	if len(lang) > maxReplyLang {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reply_lang is too long"})
		return
	}
	if strings.ContainsAny(lang, "\n\r") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reply_lang cannot contain a line break"})
		return
	}

	dic, err := config.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	dic["reply_lang"] = lang
	if err := config.Write(dic); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	filesystem.ConfigReplyLang = lang

	c.JSON(http.StatusOK, gin.H{"ok": true, "reply_lang": lang})
}

func setOutputDir(c *gin.Context) {
	var body struct {
		OutputDir *string `json:"output_dir"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.OutputDir == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "output_dir is required"})
		return
	}

	raw := strings.TrimSpace(*body.OutputDir)
	if strings.ContainsAny(raw, "\n\r") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "output_dir cannot contain a line break"})
		return
	}
	resolved, err := filesystem.ResolveOutputDir(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dic, err := config.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	dic["output_dir"] = raw
	if err := config.Write(dic); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	filesystem.ConfigOutputDir = raw

	c.JSON(http.StatusOK, gin.H{"ok": true, "output_dir": raw, "resolved": resolved})
}
