package routes

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pardnchiu/agenvoy/internal/runtime/routes/handler"
	completionsHandler "github.com/pardnchiu/agenvoy/internal/runtime/routes/handler/chatCompletions"
)

func New() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors())

	r.POST("/v1/chat/completions", completionsHandler.ChatCompletions())
	r.POST("/v1/send", handler.Send())
	r.GET("/v1/log", handler.StreamMultiLog())

	r.GET("/v1/tools", handler.ListTools())
	r.POST("/v1/tool/:tool_name", handler.CallTool())
	r.GET("/v1/models", handler.ListModels())
	r.POST("/v1/models", localhostOnly(), handler.AddModel())
	r.DELETE("/v1/models/*name", localhostOnly(), handler.RemoveModel())
	r.GET("/v1/model/dispatcher", localhostOnly(), handler.GetDispatcherModel())
	r.POST("/v1/model/dispatcher", localhostOnly(), handler.SetDispatcherModel())
	r.GET("/v1/model/summary", localhostOnly(), handler.GetSummaryModel())
	r.POST("/v1/model/summary", localhostOnly(), handler.SetSummaryModel())

	r.GET("/v1/sessions", handler.ListSessions())
	r.POST("/v1/session", localhostOnly(), handler.CreateSession())
	r.PUT("/v1/session", localhostOnly(), handler.UpdateSession())
	r.DELETE("/v1/session", localhostOnly(), handler.DeleteSession())
	r.POST("/v1/session/:session_id/model", handler.SetSessionModel())
	r.GET("/v1/session/:session_id/status", handler.GetSessionStatus())
	r.GET("/v1/session/:session_id/log", handler.StreamSessionLog())
	r.POST("/v1/session/:session_id/event", localhostOnly(), handler.PublishSessionEvent())
	r.GET("/v1/session/:session_id/pending", handler.ListSessionPending())
	r.GET("/v1/session/:session_id/pending/:task_hash/questions", handler.GetSessionPendingQuestions())
	r.POST("/v1/session/:session_id/pending/:task_hash/resume", handler.ResumeSessionPending())
	r.GET("/v1/session/:session_id/persona", localhostOnly(), handler.GetSessionPersona())
	r.POST("/v1/session/:session_id/persona", localhostOnly(), handler.SetSessionPersona())

	r.GET("/v1/file", localhostOnly(), handler.GetFile())
	r.PUT("/v1/file", localhostOnly(), handler.PutFile())

	r.GET("/v1/key", localhostOnly(), handler.GetKey())
	r.DELETE("/v1/key", localhostOnly(), handler.DeleteKey())
	r.GET("/v1/keys", localhostOnly(), handler.ListKeys())
	r.POST("/v1/keys", localhostOnly(), handler.SetKey())

	r.GET("/v1/providers", localhostOnly(), handler.ListProviders())
	r.GET("/v1/provider/:provider/check", localhostOnly(), handler.CheckProviderKey())
	r.POST("/v1/provider/:provider/key", localhostOnly(), handler.AddProviderKey())
	r.GET("/v1/provider/:provider/oauth", localhostOnly(), handler.ProviderOAuth())
	r.GET("/v1/provider/:provider/models", localhostOnly(), handler.ListProviderModels())

	r.GET("/v1/mcp", localhostOnly(), handler.ListMcpServers())
	r.POST("/v1/mcp", localhostOnly(), handler.SetMcpServer())
	r.POST("/v1/mcp/remove", localhostOnly(), handler.RemoveMcpServer())
	r.GET("/v1/mcp/status", localhostOnly(), handler.McpStatus())
	r.GET("/v1/mcp/health", localhostOnly(), handler.McpHealth())
	r.POST("/v1/mcp/reconnect", localhostOnly(), handler.McpReconnect())

	r.GET("/v1/schedule/*skill", localhostOnly(), handler.GetScheduleSkill())

	r.GET("/v1/cron", localhostOnly(), handler.ListCrons())
	r.DELETE("/v1/cron", localhostOnly(), handler.DeleteCron())
	r.POST("/v1/cron/run", localhostOnly(), handler.RunCron())

	r.GET("/v1/task", localhostOnly(), handler.ListTasks())
	r.DELETE("/v1/task", localhostOnly(), handler.DeleteTask())
	r.POST("/v1/task/run", localhostOnly(), handler.RunTask())

	return r
}

var allowedOrigins = map[string]bool{
	"https://web.agenvoy.com":                 true,
	"https://agenvoy-board.pardn.workers.dev": true,
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
			c.Header("Access-Control-Allow-Private-Network", "true")
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
		}
		c.Next()
	}
}

func localhostOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			host = c.Request.RemoteAddr
		}
		switch host {
		case "127.0.0.1", "::1":
			c.Next()
		default:
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "localhost only", "type": "forbidden"}})
		}
	}
}
