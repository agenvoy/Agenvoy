package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/go-pkg/utils"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/tools"
)

func ListMCPTools() gin.HandlerFunc {
	return func(c *gin.Context) {
		workDir, _ := os.UserHomeDir()
		executor, err := tools.NewExecutor(workDir, "api-"+utils.UUID(), agents.Scanner())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		type toolItem struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
		}

		items := make([]toolItem, 0, len(executor.Tools))
		for _, t := range executor.Tools {
			if !strings.HasPrefix(t.Function.Name, "mcp__") {
				continue
			}
			items = append(items, toolItem{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			})
		}

		c.JSON(http.StatusOK, gin.H{"tools": items})
	}
}
