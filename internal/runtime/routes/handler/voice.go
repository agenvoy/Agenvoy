package handler

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"

	"github.com/pardnchiu/agenvoy/configs"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
)

const (
	voiceKeyName         = "GEMINI_API_KEY"
	voiceModel           = "models/gemini-3.8-live"
	voiceLiveEndpoint    = "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent"
	voiceDialTimeout     = 10 * time.Second
	voiceCloseTimeout    = time.Second
	voiceCloseMissingKey = 4401
	voiceCloseReasonMax  = 120
)

var voiceUpgrader = websocket.Upgrader{}

var voiceTools = []any{
	map[string]any{
		"functionDeclarations": []any{
			map[string]any{
				"name":        "send_to_chat",
				"description": "Sends one instruction to the main Agenvoy chat agent, which can search the web, fetch live data, read and change files, run the shell and work in the repository. Its steps and reply text come back as [agent progress] messages, the end as [agent finished]. Use when the operator wants something done (幫我查 / 幫我改 / 跑一下) or asks for live or project facts you cannot know (最新 / 現在 / 這個檔案). Small talk, general knowledge, what the chat already shows, or an unclear referent → answer or ask yourself, no call.",
				"parameters": map[string]any{
					"type": "OBJECT",
					"properties": map[string]any{
						"said": map[string]any{
							"type":        "STRING",
							"description": "The operator's own sentence, kept word for word and completed from this conversation so it stands alone: the agent never heard this room, so a bare follow-up reaches it with no subject. 那明天呢 → 台北明天天氣呢; 那個 / 它 → what it refers to; a misheard word fixed. Add only what the conversation already states, never a request they did not make.",
						},
					},
					"required": []string{"said"},
				},
			},
			map[string]any{
				"name":        "read_history",
				"description": "Full transcript of this chat: every operator and agent message in order. Use for 剛剛說了什麼 / 之前查到多少 / 幫我總結, or whenever the answer may already be in the chat. Work that is still running → read_task.",
			},
			map[string]any{
				"name":        "read_task",
				"description": "The main agent's current task: the request, its reasoning and tool trace, the text produced so far, and whether it has finished. Use for 現在在做什麼 / 做到哪了 / 好了沒 / 為什麼這麼久. Earlier finished replies → read_history.",
			},
		},
	},
}

func voiceSetup() map[string]any {
	prompt := strings.ReplaceAll(filesystem.ApplyReplyLang(configs.VoicePrompt), "{{.ReplyLanguage}}", filesystem.ReplyLangDirective())
	return map[string]any{
		"setup": map[string]any{
			"model":                    voiceModel,
			"generationConfig":         map[string]any{"responseModalities": []string{"AUDIO"}},
			"systemInstruction":        map[string]any{"parts": []any{map[string]any{"text": strings.TrimSpace(prompt)}}},
			"tools":                    voiceTools,
			"inputAudioTranscription":  map[string]any{},
			"outputAudioTranscription": map[string]any{},
		},
	}
}

func VoiceLive() gin.HandlerFunc {
	return func(c *gin.Context) {
		client, err := voiceUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer client.Close()

		key := strings.TrimSpace(keychain.Get(voiceKeyName))
		if key == "" {
			voiceClose(client, voiceCloseMissingKey, voiceKeyName)
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), voiceDialTimeout)
		upstream, _, err := websocket.DefaultDialer.DialContext(ctx, voiceLiveEndpoint+"?key="+url.QueryEscape(key), nil)
		cancel()
		if err != nil {
			voiceClose(client, websocket.CloseInternalServerErr, "live dial failed")
			return
		}
		defer upstream.Close()

		if err := upstream.WriteJSON(voiceSetup()); err != nil {
			voiceClose(client, websocket.CloseInternalServerErr, "live setup failed")
			return
		}

		done := make(chan struct{}, 2)
		go voiceRelay(upstream, client, done)
		go voiceRelay(client, upstream, done)
		<-done
	}
}

func voiceRelay(dst, src *websocket.Conn, done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	for {
		kind, raw, err := src.ReadMessage()
		if err != nil {
			code, reason := websocket.CloseNormalClosure, ""
			var closed *websocket.CloseError
			if errors.As(err, &closed) && closed.Code != websocket.CloseNoStatusReceived && closed.Code != websocket.CloseAbnormalClosure {
				code, reason = closed.Code, closed.Text
			}
			voiceClose(dst, code, reason)
			return
		}
		if err := dst.WriteMessage(kind, raw); err != nil {
			return
		}
	}
}

func voiceClose(conn *websocket.Conn, code int, reason string) {
	if len(reason) > voiceCloseReasonMax {
		reason = reason[:voiceCloseReasonMax]
	}
	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(voiceCloseTimeout))
}
