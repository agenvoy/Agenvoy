package handler

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/agents"
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
