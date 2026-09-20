package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	audioTool "github.com/pardnchiu/agenvoy/internal/tools/external/audio"

	go_bot_discord "github.com/pardnchiu/go-bot/core/discord"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot"
	sessionDiscord "github.com/pardnchiu/agenvoy/internal/session/discord"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

func channelName(in go_bot_discord.Input) string {
	if in.ChannelName != "" {
		return in.ChannelName
	}
	return in.Username
}

func recordChatter(in go_bot_discord.Input, content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}

	sessionID, err := sessionDiscord.New(in.GuildID, in.ChannelID, in.UserID)
	if err != nil {
		slog.Debug("sessionDiscord.New (chatter)",
			slog.String("channel", channelName(in)),
			slog.String("error", err.Error()))
		return
	}

	username := in.Username
	if username == "" {
		username = "unknown"
	}
	if err := sessionHistory.Append(sessionID, []sessionHistory.Record{{
		Role:    "user",
		Content: content,
		SendAt:  time.Now().UnixNano(),
		Sender:  username,
	}}); err != nil {
		slog.Debug("sessionHistory.Append (chatter)",
			slog.String("channel", channelName(in)),
			slog.String("error", err.Error()))
	}
}

func run(ctx context.Context, b *Bot, in go_bot_discord.Input) error {
	if b.listener != nil && b.listener.IsAwaitingPrompt(in.ChannelID, in.MessageID) {
		if b.listener.OnCallback(ctx, in.ChannelID, in.MessageID, in.Text, in.CallbackPicks) {
			return nil
		}
	}

	content := strings.TrimSpace(in.Text)
	hasAttachment := len(in.Attachments) > 0
	if content == "" && !hasAttachment {
		return nil
	}

	_, hasPending := pending.Get(in.ChannelID)
	if in.GuildID != "" && !hasPending {
		botID := b.client.Status().UserID
		mentioned := false
		if in.Raw != nil && in.Raw.Message != nil {
			for _, u := range in.Raw.Message.Mentions {
				if u != nil && u.ID == botID {
					mentioned = true
					break
				}
			}
		}
		if !mentioned {
			recordChatter(in, content)
			return nil
		}
		content = strings.ReplaceAll(content, fmt.Sprintf("<@%s>", botID), "")
		content = strings.ReplaceAll(content, fmt.Sprintf("<@!%s>", botID), "")
		content = strings.TrimSpace(content)
		if content == "" && !hasAttachment {
			return nil
		}
	}

	if !utils.IsAuthorized(filesystem.DiscordAuthPath, in.ChannelID) {
		deleteMsg := func(msgID, label string) {
			if msgID == "" {
				return
			}
			if err := b.client.Delete(ctx, in.ChannelID, msgID); err != nil {
				slog.Debug("github.com/pardnchiu/go-bot/core/discord Bot.client.Delete",
					slog.String("label", label),
					slog.String("channel", channelName(in)),
					slog.String("msg", msgID),
					slog.String("hint", "grant Manage Messages to bot role if 50013"),
					slog.String("error", err.Error()))
			}
		}

		if p, ok := pending.Get(in.ChannelID); ok {
			if content == p.Code {
				if err := authorizeChannel(in); err != nil {
					return fmt.Errorf("authorizeChannel: %w", err)
				}
				pending.Clear(in.ChannelID)
				deleteMsg(p.PromptMsgID, "prompt")
				if in.MessageID != p.PromptMsgID {
					deleteMsg(in.MessageID, "code")
				}
				return nil
			}
			deleteMsg(p.PromptMsgID, "prompt")
			if in.MessageID != p.PromptMsgID {
				deleteMsg(in.MessageID, "unverified")
			}
		} else {
			deleteMsg(in.MessageID, "unverified")
		}

		code, err := utils.GenerateAuthCode()
		if err != nil {
			return fmt.Errorf("utils.GenerateAuthCode: %w", err)
		}
		slog.Info("Discord Verification Code",
			slog.String("name", channelName(in)),
			slog.String("code", code))
		exec.NotifyAdminCode(ctx, code, "Discord "+channelName(in))
		prompt, err := b.client.SendInput(ctx, in.ChannelID, "", "Enter the 6-digit verification code printed in the daemon log.")
		if err != nil {
			slog.Warn("github.com/pardnchiu/go-bot/core/discord Bot.client.SendInput",
				slog.String("channel", channelName(in)),
				slog.String("error", err.Error()))
			return nil
		}
		promptID := ""
		if prompt != nil {
			promptID = prompt.ID
		}
		pending.Set(in.ChannelID, code, promptID)
		return nil
	}

	if hasAttachment {
		if hasVoiceAttachment(in) && !audioTool.STTEnabled() {
			_, _ = b.client.Send(ctx, in.ChannelID, in.MessageID, "No speech-to-text model selected · pick one with `/model stt` first.")
			return nil
		}
		attachments := saveAttachments(ctx, b, in)
		transcripts, paths, err := chatbot.TranscribeSavedAttachments(ctx, attachments)
		if err != nil {
			slog.Debug("transcribeSavedAttachments",
				slog.String("channel", channelName(in)),
				slog.String("error", err.Error()))
			_, _ = b.client.Send(ctx, in.ChannelID, in.MessageID, fmt.Sprintf("⚠️ Voice transcription failed\n`%s`", err.Error()))
			return nil
		}
		if len(transcripts) > 0 || len(paths) > 0 {
			var lines []string
			if content != "" {
				lines = append(lines, content)
			}
			lines = append(lines, transcripts...)
			if len(paths) > 0 {
				lines = append(lines, "[Discord attachments]")
				for _, p := range paths {
					lines = append(lines, "- "+p)
				}
			}
			content = strings.Join(lines, "\n")
		}
	}

	if content == "" {
		return nil
	}

	discordSessionID, err := sessionDiscord.New(in.GuildID, in.ChannelID, in.UserID)
	if err != nil {
		return fmt.Errorf("github.com/pardnchiu/agenvoy/internal/session GetDiscordSession: %w", err)
	}

	reply := b.newReply(ctx, in.ChannelID, channelName(in), discordSessionID, in.MessageID)
	reply.Status("thinking...")

	return reply.Run(ctx, exec.ExecuteMeta{
		Content:        content,
		Input:          content,
		AllowAll:       false,
		ReplyMessageID: in.MessageID,
		Sender:         in.Username,
		SessionID:      discordSessionID,
	})
}
