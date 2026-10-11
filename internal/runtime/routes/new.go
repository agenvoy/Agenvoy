package routes

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/runtime/routes/handler"
	completionsHandler "github.com/pardnchiu/agenvoy/internal/runtime/routes/handler/chatCompletions"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

func New() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors())

	// * OpenAI compatible API
	r.POST("/v1/chat/completions", completionsHandler.ChatCompletions())
	r.GET("/v1/models", handler.ListModels())
	r.GET("/v1/models/*id", handler.GetModel())

	// * TUI API
	r.GET("/v1/log", localhostOnly(), handler.StreamDaemonLog())

	r.GET("/v1/providers", localhostOnly(), handler.ListProviders())
	r.GET("/v1/provider/:provider/console", localhostOnly(), handler.ProviderConsole())

	r.GET("/v1/roles", localhostOnly(), handler.ListRoles())
	r.GET("/v1/role/*name", localhostOnly(), handler.GetRole())
	r.POST("/v1/role", localhostOnly(), handler.CreateRole())
	r.PATCH("/v1/role", localhostOnly(), handler.UpdateRole())

	r.DELETE("/v1/channel/:channel/chat", localhostOnly(), handler.DeleteChannelChat())

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
			"message": "not found",
			"type":    "not_found",
		}})
	})

	return r
}

func allowOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	switch parsed.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin := c.GetHeader("Origin"); origin != "" && allowOrigin(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, "+completionsHandler.AgentHeader)
			c.Header("Access-Control-Allow-Private-Network", "true")
		}
		c.Header("Vary", "Origin")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func localhostOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !utils.IsLoopback(c.Request.RemoteAddr) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "localhost only", "type": "forbidden"}})
			return
		}
		c.Next()
	}
}
