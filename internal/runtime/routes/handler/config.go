package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/startup"
	"github.com/pardnchiu/agenvoy/internal/session/config"
)

func startupEnabled() bool {
	if recorded, ok := startup.Recorded(); ok {
		return recorded
	}
	return startup.Enabled()
}

func GetStartup() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"enabled": startupEnabled(), "installed": startup.Enabled()})
	}
}

func SetStartup() gin.HandlerFunc {
	return func(c *gin.Context) {
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
		c.JSON(http.StatusOK, gin.H{"ok": true, "enabled": startupEnabled(), "installed": startup.Enabled(), "detail": detail})
	}
}

const maxReplyLang = 64

func GetSystemConfig() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"reply_lang": filesystem.ReplyLang,
			"languages":  filesystem.ReplyLangOptions(),
		})
	}
}

func SetSystemConfig() gin.HandlerFunc {
	return func(c *gin.Context) {
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
		filesystem.ReplyLang = lang

		c.JSON(http.StatusOK, gin.H{"ok": true, "reply_lang": lang})
	}
}
