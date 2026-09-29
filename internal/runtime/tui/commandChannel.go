package tui

import (
	"context"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot/discord"
	"github.com/pardnchiu/agenvoy/internal/runtime/chatbot/line"
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

var channelTabs = []string{"telegram", "discord", "line"}

func (t TUI) commandChannel(parts []string) (TUI, tea.Cmd, bool) {
	tab := 0
	if len(parts) > 1 {
		if len(parts) > 2 {
			switch parts[1] {
			case "telegram":
				return t.commandTelegram(parts[1:])
			case "discord":
				return t.commandDiscord(parts[1:])
			case "line":
				return t.commandLine(parts[1:])
			}
		}
		tab = max(slices.Index(channelTabs, parts[1]), 0)
	}

	popup := &Popup{
		kind:  popupSingleSelect,
		title: "/channel",
		tabs:  channelTabs,
	}
	popup.onConfirm = func(chosen string) any {
		if popup.kind == popupText {
			value := strings.TrimSpace(chosen)
			switch channelTabs[popup.tabIdx] {
			case "discord":
				return DiscordTokenSubmit{token: value}
			case "line":
				return LineSecretSubmit{secret: value}
			}
			return TelegramTokenSubmit{token: value}
		}
		{
			channel, action, _ := strings.Cut(chosen, ":")
			switch action {
			case "enable":
				return ChannelSelect{channel: channel}
			case "disable":
				switch channel {
				case "discord":
					return DiscordAction{action: "disable"}
				case "line":
					return LineAction{action: "disable"}
				}
				return TelegramAction{action: "disable"}
			}
			return nil
		}
	}
	popup.onDelete = func(chosen string) any {
		channel, id, _ := strings.Cut(chosen, ":")
		if id == "" || id == "enable" || id == "disable" {
			return nil
		}
		return ChannelRevokePick{channel: channel, id: id, name: chatName(channel, id)}
	}
	popup.onTab = func(p *Popup) tea.Cmd {
		fillChannelTab(p, channelTabs[p.tabIdx])
		return nil
	}
	popup.tabIdx = tab
	popup.onTab(popup)
	t.popup = popup
	return t, nil, true
}

func channelEnabled(channel string) bool {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return false
	}
	switch channel {
	case "discord":
		return cfg.DiscordEnabled && keychain.Get(discord.Key) != ""
	case "line":
		return cfg.LineEnabled && keychain.Get(line.SecretKey) != "" && keychain.Get(line.TokenKey) != ""
	}
	return cfg.TelegramEnabled && keychain.Get(telegram.Key) != ""
}

func chatName(channel, id string) string {
	for _, one := range utils.ListChats(channelAuthPath(channel)) {
		if one.ID == id {
			return strings.TrimSpace(one.Name)
		}
	}
	return ""
}

func fillChannelTab(p *Popup, channel string) {
	p.cursor = 0
	p.styledLines = nil
	if !channelEnabled(channel) {
		field, source := "token", "@BotFather"
		switch channel {
		case "discord":
			source = "Discord Developer Portal"
		case "line":
			field, source = "channel secret", "LINE Developers Console"
		}
		p.kind = popupText
		p.multiline = false
		p.input = newPopupInput("", false)
		p.subtitle = "not connected  enter the " + field + " from " + source
		p.options, p.values = nil, nil
		return
	}
	p.kind = popupSingleSelect

	entries := utils.ListChats(channelAuthPath(channel))
	prefix := channelPrefix(channel)
	options := make([]string, 0, len(entries)+2)
	values := make([]string, 0, len(entries)+2)
	for _, one := range entries {
		options = append(options, adminChannelLabel(prefix, one))
		values = append(values, channel+":"+one.ID)
	}
	if len(entries) > 0 {
		options = append(options, "")
		values = append(values, "")
	}
	options = append(options, "disable")
	values = append(values, channel+":disable")

	p.subtitle = "authorized chats  d revokes the highlighted one"
	p.options, p.values = options, values
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
	switch channel {
	case "discord":
		return filesystem.DiscordAuthPath
	case "line":
		return filesystem.LineAuthPath
	}
	return filesystem.TelegramAuthPath
}

func channelPrefix(channel string) string {
	switch channel {
	case "discord":
		return "dc"
	case "line":
		return "ln"
	}
	return "tg"
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
