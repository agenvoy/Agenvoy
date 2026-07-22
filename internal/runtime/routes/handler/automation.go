package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/filesystem/skill"
	"github.com/pardnchiu/agenvoy/internal/runtime"
)

func GetScheduleSkill() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.TrimPrefix(c.Param("skill"), "/")
		name = strings.TrimSpace(name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "skill is required"})
			return
		}
		body, err := skill.GetSchedule(name)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"skill": name, "body": body})
	}
}

func ListCrons() gin.HandlerFunc {
	return func(c *gin.Context) {
		crons, err := runtime.LoadCrons()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"crons": crons})
	}
}

func RemoveCron() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Skill string `json:"skill"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		skill := strings.TrimSpace(body.Skill)
		if skill == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "skill is required"})
			return
		}
		removed, err := runtime.RemoveCron(skill)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if removed == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "cron not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "removed": removed})
	}
}

func RunCron() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			SessionID string `json:"session_id"`
			Skill     string `json:"skill"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sessionID := strings.TrimSpace(body.SessionID)
		skill := strings.TrimSpace(body.Skill)
		if sessionID == "" || skill == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and skill are required"})
			return
		}
		go runtime.Fire(sessionID, skill)
		c.JSON(http.StatusAccepted, gin.H{"ok": true, "started": true})
	}
}

func ListTasks() gin.HandlerFunc {
	return func(c *gin.Context) {
		tasks, err := runtime.LoadTasks()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tasks": tasks})
	}
}

func RemoveTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Skill string `json:"skill"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		skill := strings.TrimSpace(body.Skill)
		if skill == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "skill is required"})
			return
		}
		removed, err := runtime.RemoveTask(skill)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if removed == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "removed": removed})
	}
}

func RunTask() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			SessionID string `json:"session_id"`
			Skill     string `json:"skill"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sessionID := strings.TrimSpace(body.SessionID)
		skill := strings.TrimSpace(body.Skill)
		if sessionID == "" || skill == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and skill are required"})
			return
		}
		go runtime.Fire(sessionID, skill)
		c.JSON(http.StatusAccepted, gin.H{"ok": true, "started": true})
	}
}
