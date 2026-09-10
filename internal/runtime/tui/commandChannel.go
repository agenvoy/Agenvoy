package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot/discord"
	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot/telegram"
	"github.com/pardnchiu/agenvoy/internal/runtime/daemon"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	"github.com/pardnchiu/agenvoy/internal/utils"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
)

const channelRevokeTimeout = 10 * time.Second

type ChannelSelect struct {
	channel string
}

func (t TUI) commandChannel(parts []string) (TUI, tea.Cmd, bool) {
	if len(parts) > 1 {
		switch parts[1] {
		case "telegram":
			return t.commandTelegram(parts[1:])
		case "discord":
			return t.commandDiscord(parts[1:])
		case "admin":
			return t.commandAdminChannel(parts[1:])
		}
	}

	cfg, err := config.Load()
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}

	state := func(enabled bool, key string) string {
		if enabled && keychain.Get(key) != "" {
			return systemStyle.Render("[enabled]")
		}
		return ""
	}

	values := []string{"admin", "telegram", "discord"}
	details := []string{
		hintStyle.Render("relay new-chat verification codes"),
		state(cfg.TelegramEnabled, telegram.Key),
		state(cfg.DiscordEnabled, discord.Key),
	}

	t.popup = &Popup{
		kind:    popupSingleSelect,
		title:   "Channel",
		options: optionColumn(values, details),
		values:  values,
		onConfirm: func(chosen string) any {
			return ChannelSelect{channel: chosen}
		},
	}
	return t, nil, true
}

type ChannelRevokePick struct {
	channel string
	id      string
	name    string
}

type ChannelRevokeConfirm struct {
	channel string
	id      string
	label   string
	yes     bool
}

type ChannelRevokeDone struct {
	channel string
	name    string
	err     error
}

func channelAuthPath(channel string) string {
	if channel == "discord" {
		return filesystem.DiscordAuthPath
	}
	return filesystem.TelegramAuthPath
}

func channelPrefix(channel string) string {
	if channel == "discord" {
		return "dc"
	}
	return "tg"
}

func (t TUI) openChannelMenu(channel, title string, disable func() any) (TUI, tea.Cmd) {
	entries := utils.ListChats(channelAuthPath(channel))
	prefix := channelPrefix(channel)

	options := make([]string, 0, len(entries)+1)
	values := make([]string, 0, len(entries)+1)
	names := make(map[string]string, len(entries))
	for _, one := range entries {
		options = append(options, adminChannelLabel(prefix, one))
		values = append(values, one.ID)
		names[one.ID] = strings.TrimSpace(one.Name)
	}
	options = append(options, "(disable "+channel+")")
	values = append(values, "disable")

	t.popup = &Popup{
		kind:       popupSingleSelect,
		title:      title,
		subtitle:   "authorized chats  d revokes the highlighted one",
		options:    options,
		values:     values,
		maxVisible: cmdSelectorMaxVisible,
		onConfirm: func(chosen string) any {
			if chosen == "disable" {
				return disable()
			}
			return nil
		},
		onDelete: func(chosen string) any {
			if chosen == "disable" {
				return nil
			}
			return ChannelRevokePick{channel: channel, id: chosen, name: names[chosen]}
		},
	}
	return t, nil
}

func (t TUI) openChannelRevokeConfirm(msg ChannelRevokePick) (TUI, tea.Cmd) {
	label := msg.id
	if msg.name != "" {
		label = msg.name + " (" + msg.id + ")"
	}

	t.popup = &Popup{
		kind:     popupSingleSelect,
		title:    "Revoke " + label + " ?",
		subtitle: "it stops receiving replies until it verifies again",
		options:  []string{"No", "Yes"},
		values:   []string{"no", "yes"},
		onConfirm: func(chosen string) any {
			return ChannelRevokeConfirm{channel: msg.channel, id: msg.id, label: label, yes: chosen == "yes"}
		},
		onCancel: func() any {
			return ChannelRevokeConfirm{channel: msg.channel, id: msg.id, label: label}
		},
	}
	return t, nil
}

func revokeChannelChat(channel, id, label string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), channelRevokeTimeout)
		defer cancel()

		_, err := daemon.Delete[map[string]any](ctx, "/v1/channel/"+channel+"/chat", map[string]any{"id": id})
		return ChannelRevokeDone{channel: channel, name: label, err: err}
	}
}
