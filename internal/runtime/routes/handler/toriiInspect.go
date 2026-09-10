package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/agents/exec/memory"
)

func ListErrorMemory() gin.HandlerFunc {
	return func(c *gin.Context) {
		tool := strings.TrimSpace(c.Query("tool"))
		keyword := strings.TrimSpace(c.Query("keyword"))

		if keyword == "" {
			limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
			if err != nil {
				limit = 50
			}
			c.JSON(http.StatusOK, gin.H{"records": memory.List(tool, limit)})
			return
		}

		limit, err := strconv.Atoi(c.DefaultQuery("limit", "16"))
		if err != nil {
			limit = 16
		}

		result := memory.Search(c.Request.Context(), tool, keyword, limit)
		if result == "NONE" {
			c.JSON(http.StatusOK, gin.H{"records": []memory.Record{}})
			return
		}

		var records []memory.Record
		if err := json.Unmarshal([]byte(result), &records); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"records": records})
	}
}

func UpdateErrorMemory() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			ID     string `json:"id"`
			Action string `json:"action"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		body.ID = strings.TrimSpace(body.ID)
		body.Action = strings.TrimSpace(body.Action)
		if body.ID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
			return
		}
		if body.Action == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "action is required"})
			return
		}

		record, err := memory.UpdateAction(c.Request.Context(), body.ID, body.Action)
		if errors.Is(err, memory.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"record": record})
	}
}
