package handler

import (
	"encoding/json"
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

		if tool == "" && keyword == "" {
			limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
			if err != nil {
				limit = 50
			}
			c.JSON(http.StatusOK, gin.H{"records": memory.List(limit)})
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
