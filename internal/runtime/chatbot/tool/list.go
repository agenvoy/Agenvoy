package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

func registListChatbot() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "list_chatbot",
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `[system-default] List authorized chats for the specified platform (Telegram or Discord).`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"platform": platformParam(),
			},
			"required": []string{"platform"},
		},
		Handler: func(_ context.Context, _ *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Platform string `json:"platform"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			platform, err := parsePlatform(strings.TrimSpace(params.Platform))
			if err != nil {
				return "", err
			}

			var authPath string
			switch platform {
			case platformTelegram:
				authPath = filesystem.TelegramAuthPath
			case platformDiscord:
				authPath = filesystem.DiscordAuthPath
			}

			entries := utils.ListChats(authPath)
			if entries == nil {
				entries = []utils.ChatEntry{}
			}
			raw, err := json.Marshal(entries)
			if err != nil {
				return "", fmt.Errorf("json.Marshal: %w", err)
			}
			return string(raw), nil
		},
	})
}
