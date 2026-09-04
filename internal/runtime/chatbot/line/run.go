package line

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot"
	go_bot_line "github.com/pardnchiu/go-bot/line"

	"github.com/pardnchiu/agenvoy/internal/agents"
	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot"
	"github.com/pardnchiu/agenvoy/internal/runtime/pubsub"
	sessionManager "github.com/pardnchiu/agenvoy/internal/session"
	sessionHistory "github.com/pardnchiu/agenvoy/internal/session/history"
	sessionLog "github.com/pardnchiu/agenvoy/internal/session/log"
	"github.com/pardnchiu/agenvoy/internal/tools"
	audioTool "github.com/pardnchiu/agenvoy/internal/tools/external/audio"
	"github.com/pardnchiu/agenvoy/internal/utils"
)

func sourceID(in go_bot_line.Input) string {
	switch {
	case in.GroupID != "":
		return in.GroupID
	case in.RoomID != "":
		return in.RoomID
	default:
		return in.UserID
	}
}

func inputHasAttachment(in go_bot_line.Input) bool {
	return in.MessageType != "" && in.MessageType != "text" && in.MessageID != ""
}

func inputHasVoice(in go_bot_line.Input) bool {
	return in.MessageID != "" && (in.MessageType == "audio" || in.MessageType == "video")
}

func sourceName(in go_bot_line.Input) string {
	if in.Username != "" {
		return in.Username
	}
	switch {
	case in.GroupID != "":
		return "group:" + in.GroupID
	case in.RoomID != "":
		return "room:" + in.RoomID
	default:
		return "user:" + in.UserID
	}
}

func isMentioned(in go_bot_line.Input, botID string) bool {
	if in.Raw == nil || botID == "" {
		return false
	}
	tm, ok := in.Raw.Message.(*linebot.TextMessage)
	if !ok || tm.Mention == nil {
		return false
	}
	for _, m := range tm.Mention.Mentionees {
		if m == nil {
			continue
		}
		if m.Type == linebot.MentionedTargetTypeAll || m.UserID == botID {
			return true
		}
	}
	return false
}

func recordChatter(in go_bot_line.Input, content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}

	sessionID, err := sessionManager.GetLineSession(in.UserID, in.GroupID, in.RoomID)
	if err != nil {
		slog.Warn("sessionManager.GetLineSession (chatter)",
			slog.String("source", sourceName(in)),
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
		slog.Warn("sessionHistory.Append (chatter)",
			slog.String("source", sourceName(in)),
			slog.String("error", err.Error()))
	}
}

func run(ctx context.Context, b *Bot, in go_bot_line.Input, attachInputs []go_bot_line.Input) error {
	content := strings.TrimSpace(in.Text)
	if content == "" {
		for _, ai := range attachInputs {
			if t := strings.TrimSpace(ai.Text); t != "" {
				content = t
				break
			}
		}
	}
	hasAttachment := slices.ContainsFunc(attachInputs, inputHasAttachment)
	if content == "" && !hasAttachment {
		return nil
	}

	target := sourceID(in)
	if target == "" {
		return fmt.Errorf("no source id")
	}

	_, hasVerifyPending := pending.Get(target)
	if (in.GroupID != "" || in.RoomID != "") && !hasVerifyPending {
		status := b.client.Status()
		if !isMentioned(in, status.UserID) {
			recordChatter(in, content)
			return nil
		}
		if status.DisplayName != "" {
			content = strings.TrimSpace(strings.ReplaceAll(content, "@"+status.DisplayName, ""))
			if content == "" && !hasAttachment {
				return nil
			}
		}
	}

	if !utils.IsAuthorized(filesystem.LineAuthPath, target) {
		if p, ok := pending.Get(target); ok && content == p.Code {
			if err := authorizeSource(target, in.Username); err != nil {
				return fmt.Errorf("authorizeSource: %w", err)
			}
			pending.Clear(target)
			if _, err := b.client.Send(ctx, target, "verified, you can start the conversation."); err != nil {
				slog.Warn("github.com/pardnchiu/go-bot/line Bot.Send (verified)",
					slog.String("source", target),
					slog.String("error", err.Error()))
			}
			return nil
		}

		code, err := utils.GenerateAuthCode()
		if err != nil {
			return fmt.Errorf("utils.GenerateAuthCode: %w", err)
		}
		slog.Info("LINE Verification Code",
			slog.String("name", sourceName(in)),
			slog.String("code", code))
		exec.NotifyAdminCode(ctx, code, "LINE "+sourceName(in))
		pending.Set(target, code, "")
		if _, err := b.client.Send(ctx, target, "please enter the 6-digit verification code to enable the conversation."); err != nil {
			slog.Warn("github.com/pardnchiu/go-bot/line Bot.Send (verify prompt)",
				slog.String("source", target),
				slog.String("error", err.Error()))
		}
		return nil
	}

	if hasAttachment {
		if slices.ContainsFunc(attachInputs, inputHasVoice) && !audioTool.STTEnabled() {
			if _, err := b.client.Send(ctx, target, "⚠️ no speech-to-text model selected · pick one with /model stt first."); err != nil {
				slog.Warn("github.com/pardnchiu/go-bot/line Bot.Send (voice disabled)",
					slog.String("source", target),
					slog.String("error", err.Error()))
			}
			return nil
		}

		pathName := map[string]string{}
		var attachments []chatbot.SavedAttachment
		for _, ai := range attachInputs {
			if !inputHasAttachment(ai) {
				continue
			}
			path, err := b.client.Save(ctx, ai.MessageID, filesystem.DownloadDir)
			if err != nil {
				slog.Warn("github.com/pardnchiu/go-bot/line Bot.Save",
					slog.String("source", target),
					slog.String("messageType", ai.MessageType),
					slog.String("error", err.Error()))
				continue
			}
			if path == "" {
				continue
			}
			if ai.FileName != "" {
				pathName[path] = ai.FileName
			}
			attachments = append(attachments, chatbot.SavedAttachment{Path: path, Transcribe: inputHasVoice(ai)})
		}

		transcripts, paths, err := chatbot.TranscribeSavedAttachments(ctx, attachments)
		if err != nil {
			slog.Warn("chatbot.TranscribeSavedAttachments",
				slog.String("source", target),
				slog.String("error", err.Error()))
			if _, sendErr := b.client.Send(ctx, target, fmt.Sprintf("⚠️ voice transcription failed\n%s", err.Error())); sendErr != nil {
				slog.Warn("github.com/pardnchiu/go-bot/line Bot.Send (transcribe failure)",
					slog.String("source", target),
					slog.String("error", sendErr.Error()))
			}
			return nil
		}
		if len(transcripts) > 0 || len(paths) > 0 {
			var lines []string
			if content != "" {
				lines = append(lines, content)
			}
			lines = append(lines, transcripts...)
			if len(paths) > 0 {
				lines = append(lines, "[LINE attachments]")
				for _, one := range paths {
					label := "- " + one
					if name := pathName[one]; name != "" {
						label += " (" + name + ")"
					}
					lines = append(lines, label)
				}
			}
			content = strings.Join(lines, "\n")
		}
	}

	if content == "" {
		if _, err := b.client.Send(ctx, target, "⚠️ failed to receive the attachment."); err != nil {
			slog.Warn("github.com/pardnchiu/go-bot/line Bot.Send (attachment failure)",
				slog.String("source", target),
				slog.String("error", err.Error()))
		}
		return nil
	}

	workDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("os.UserHomeDir: %w", err)
	}

	scanner := agents.Scanner()
	if scanner != nil {
		scanner.Scan()
	}

	sessionID, err := sessionManager.GetLineSession(in.UserID, in.GroupID, in.RoomID)
	if err != nil {
		return fmt.Errorf("github.com/pardnchiu/agenvoy/internal/session GetLineSession: %w", err)
	}
	pubsub.Pub(sessionID, agentTypes.Event{Type: agentTypes.EventUserInput, Text: content})

	primary, fallbacks, err := exec.ResolveAgent(ctx, agents.DispatcherBot(), agents.Registry(), content, false, "", sessionID)
	if err != nil {
		if _, sendErr := b.client.Send(ctx, target, fmt.Sprintf("⚠️ %s", err.Error())); sendErr != nil {
			slog.Warn("github.com/pardnchiu/go-bot/line Bot.Send (ResolveAgent error reply)",
				slog.String("source", target),
				slog.String("error", sendErr.Error()))
		}
		return fmt.Errorf("ResolveAgent: %w", err)
	}

	if primary != nil {
		agentName := strings.TrimSpace(primary.Name())
		agentResult := agentTypes.Event{Type: agentTypes.EventAgentResult, Text: agentName, Model: agentName}
		sessionLog.Record(sessionID, agentResult)
		pubsub.Pub(sessionID, agentResult)
	}

	execData := exec.ExecuteMeta{
		Agent:          primary,
		FallbackAgents: fallbacks,
		WorkDir:        workDir,
		Content:        content,
		ExcludeTools:   tools.TUIOnlyTools,
		ExcludeSkills:  tools.TUIOnlySkills,
		AllowAll:       true,
		Sender:         sourceName(in),
	}

	sess, err := getSession(ctx, in, content, execData)
	if err != nil {
		return fmt.Errorf("getSession: %w", err)
	}
	utils.EventLog("[LINE]", agentTypes.Event{}, sess.ID, content)

	events := make(chan agentTypes.Event, 128)
	// * tee into pubsub so the web view can stream this channel session live
	wrapped := pubsub.Wrap(ctx, sess.ID, events, 128)
	go func() {
		execCtx := exec.SuppressDcPush(ctx)
		if execErr := exec.Execute(execCtx, execData, sess, wrapped, execData.AllowAll); execErr != nil {
			slog.Warn("exec",
				slog.String("session", sess.ID),
				slog.String("error", execErr.Error()))
		}
		close(wrapped)
	}()

	result := utils.FormatChatbotEvent(events, "[LINE]", sess.ID, func(string) {}, func(toolName, text string) string {
		return fmt.Sprintf("%s: %s", toolName, text)
	})

	replyText, _ := utils.ExtractFileMarkers(strings.TrimSpace(result.ReplyText))
	replyText = strings.TrimSpace(replyText)
	if replyText == "" {
		return fmt.Errorf("no reply")
	}

	model := result.Done.Model
	if model == "" && primary != nil {
		model = primary.Name()
	}
	if footer := utils.FormatEventFooter(result.Done.Duration, model, result.Done.Usage); footer != "" {
		replyText = replyText + "\n\n" + footer
	}
	if len(result.ExecErrors) > 0 {
		replyText = replyText + "\n⚠️ " + strings.Join(result.ExecErrors, ", ")
	}

	for _, part := range chunk(replyText) {
		if _, err := b.client.Send(ctx, target, part); err != nil {
			slog.Warn("github.com/pardnchiu/go-bot/line Bot.Send",
				slog.String("session", sess.ID),
				slog.String("source", target),
				slog.String("error", err.Error()))
			return nil
		}
	}
	return nil
}
