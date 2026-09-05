package routes

import (
	"net/http"
	"net/url"
	"strings"

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

	pageIndex := registPage(r)

	// * OpenAI compatible API
	r.POST("/v1/chat/completions", completionsHandler.ChatCompletions())
	r.GET("/v1/models", handler.ListModels())
	r.GET("/v1/models/*id", handler.GetModel())

	// * TUI usage ()
	r.POST("/v1/session/:session_id/event", localhostOnly(), handler.PublishSessionEvent())

	// * WebUI API
	r.POST("/v1/send", handler.Send())
	r.GET("/v1/log", handler.StreamMultiLog())
	r.GET("/v1/info/version", handler.GetVersion())
	r.GET("/v1/daemon", localhostOnly(), handler.GetDaemonLog())

	r.POST("/v1/models", localhostOnly(), handler.AddModel())
	r.DELETE("/v1/models/*name", localhostOnly(), handler.RemoveModel())
	r.GET("/v1/model", localhostOnly(), handler.GetModelRouting())
	r.POST("/v1/model", localhostOnly(), handler.SetModelRouting())
	r.GET("/v1/model/audio", localhostOnly(), handler.ListAudioModels())

	r.GET("/v1/usage", localhostOnly(), handler.GetTotalUsage())

	r.GET("/v1/sessions", handler.ListSessions())
	r.POST("/v1/session", localhostOnly(), handler.CreateSession())
	r.GET("/v1/session/:session_id", localhostOnly(), handler.GetSession())
	r.POST("/v1/session/:session_id", localhostOnly(), handler.UpdateSession())
	r.DELETE("/v1/session/:session_id", localhostOnly(), handler.DeleteSession())
	r.POST("/v1/session/:session_id/memory", localhostOnly(), handler.SessionMemory())

	r.POST("/v1/session/:session_id/confirm/:once_id", handler.ResolveToolConfirm())
	r.POST("/v1/session/:session_id/cancel/:once_id", handler.CancelSessionTask())

	// * WebUI session task · pending file
	r.GET("/v1/session/:session_id/task", handler.ListSessionPending())
	r.GET("/v1/session/:session_id/task/:task_hash/questions", handler.GetSessionPendingQuestions())
	r.POST("/v1/session/:session_id/task/:task_hash/resume", handler.ResumeSessionPending())
	r.DELETE("/v1/session/:session_id/task/:task_hash", handler.DeleteSessionPending())

	// * WebUI session task · sqlite
	r.GET("/v1/session/:session_id/task/history", localhostOnly(), handler.ListTaskHistory())
	r.GET("/v1/session/:session_id/task/:task_hash/history", localhostOnly(), handler.GetTaskHistory())

	r.GET("/v1/file", localhostOnly(), handler.GetFile())
	r.PUT("/v1/file", localhostOnly(), handler.PutFile())
	r.GET("/v1/file/open", localhostOnly(), handler.OpenFile())
	r.GET("/v1/file/locate", localhostOnly(), handler.LocateFile())
	r.GET("/v1/workdir", localhostOnly(), handler.CheckWorkDir())

	r.GET("/v1/key", localhostOnly(), handler.GetKey())
	r.DELETE("/v1/key", localhostOnly(), handler.DeleteKey())
	r.GET("/v1/keys", localhostOnly(), handler.ListKeys())
	r.POST("/v1/keys", localhostOnly(), handler.SetKey())

	r.GET("/v1/providers", localhostOnly(), handler.ListProviders())
	r.GET("/v1/providers/usage", localhostOnly(), handler.ListProviderUsage())
	r.POST("/v1/provider/:provider/key", localhostOnly(), handler.AddProviderKey())
	r.GET("/v1/provider/:provider/oauth", localhostOnly(), handler.ProviderOAuth())
	r.DELETE("/v1/provider/:provider/oauth", localhostOnly(), handler.ClearProviderOAuth())
	r.GET("/v1/provider/:provider/models", localhostOnly(), handler.ListProviderModels())

	r.GET("/v1/mcp", localhostOnly(), handler.ListMcpServers())
	r.GET("/v1/mcp/tools", handler.ListMCPTools())
	r.POST("/v1/mcp", localhostOnly(), handler.SetMcpServer())
	r.POST("/v1/mcp/remove", localhostOnly(), handler.RemoveMcpServer())
	r.GET("/v1/mcp/status", localhostOnly(), handler.McpStatus())
	r.POST("/v1/mcp/reconnect", localhostOnly(), handler.McpReconnect())
	r.GET("/v1/mcp/oauth", localhostOnly(), handler.McpOAuthLogin())
	r.POST("/v1/mcp/oauth/callback", localhostOnly(), handler.McpOAuthCallback())
	r.POST("/v1/mcp/oauth/client", localhostOnly(), handler.McpOAuthClient())
	r.DELETE("/v1/mcp/oauth", localhostOnly(), handler.McpOAuthClear())

	r.GET("/v1/rules", localhostOnly(), handler.ListRules())
	r.GET("/v1/rule/*name", localhostOnly(), handler.GetRule())
	r.POST("/v1/rule", localhostOnly(), handler.CreateRule())
	r.PATCH("/v1/rule", localhostOnly(), handler.UpdateRule())
	r.DELETE("/v1/rule", localhostOnly(), handler.DeleteRule())

	r.GET("/v1/knowledges", localhostOnly(), handler.ListKnowledges())
	r.GET("/v1/knowledge/*name", localhostOnly(), handler.GetKnowledge())
	r.POST("/v1/knowledge", localhostOnly(), handler.CreateKnowledge())
	r.PATCH("/v1/knowledge", localhostOnly(), handler.UpdateKnowledge())
	r.DELETE("/v1/knowledge", localhostOnly(), handler.DeleteKnowledge())

	r.GET("/v1/skills", localhostOnly(), handler.ListSkills())
	r.GET("/v1/skill/*name", localhostOnly(), handler.GetSkill())
	r.DELETE("/v1/skill", localhostOnly(), handler.DeleteSkill())

	r.GET("/v1/schedule", localhostOnly(), handler.ListSchedules())
	r.POST("/v1/schedule", localhostOnly(), handler.CreateSchedule())
	r.PATCH("/v1/schedule", localhostOnly(), handler.UpdateSchedule())
	r.DELETE("/v1/schedule", localhostOnly(), handler.DeleteSchedule())
	r.POST("/v1/schedule/run", localhostOnly(), handler.RunSchedule())
	r.GET("/v1/schedule/*skill", localhostOnly(), handler.GetScheduleSkill())

	r.GET("/v1/allowlist", localhostOnly(), handler.GetAllowlist())
	r.POST("/v1/allowlist", localhostOnly(), handler.SetAllowlist())

	r.GET("/v1/config/startup", localhostOnly(), handler.GetStartup())
	r.POST("/v1/config/startup", localhostOnly(), handler.SetStartup())

	r.GET("/v1/torii/error", localhostOnly(), handler.ListErrorMemory())

	r.GET("/v1/channel", localhostOnly(), handler.GetChannel())
	r.POST("/v1/channel/telegram", localhostOnly(), handler.SetTelegramChannel())
	r.POST("/v1/channel/discord", localhostOnly(), handler.SetDiscordChannel())
	r.POST("/v1/channel/admin", localhostOnly(), handler.SetAdminChannel())
	r.GET("/v1/channel/:channel/chats", localhostOnly(), handler.ListChannelChats())
	r.DELETE("/v1/channel/:channel/chat", localhostOnly(), handler.DeleteChannelChat())

	r.NoRoute(localhostOnly(), func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/v1/") {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{
				"message": "not found",
				"type":    "not_found",
			}})
			return
		}
		pageIndex(c)
	})

	return r
}

var allowedOrigins = map[string]bool{
	"https://web.agenvoy.com":                 true,
	"https://agenvoy-board.pardn.workers.dev": true,
}

func allowOrigin(origin string) bool {
	if allowedOrigins[origin] {
		return true
	}
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
