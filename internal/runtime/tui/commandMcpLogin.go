package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
)

type mcpClientDraft struct {
	server string
	id     string
	secret string
}

type McpClientID struct {
	id string
}

type McpClientSecret struct {
	secret string
}

type McpClientRedirect struct {
	uri string
}

type McpOAuthInfo struct {
	url string
}

type McpOAuthDone struct {
	name string
	err  error
}

type McpOAuthPaste struct {
	server string
	url    string
}

func (t TUI) openMcpClientID(server string) (TUI, tea.Cmd) {
	t.mcpClient = &mcpClientDraft{server: server}
	t.popup = &Popup{
		kind:     popupText,
		title:    fmt.Sprintf("%s · OAuth client ID", server),
		subtitle: "from the provider console; register " + mcp.DefaultRedirectURI + " as its redirect URI",
		input:    newPopupInput("", false),
		onConfirm: func(value string) any {
			return McpClientID{id: strings.TrimSpace(value)}
		},
	}
	return t, nil
}

func (t TUI) openMcpClientSecret() (TUI, tea.Cmd) {
	t.popup = &Popup{
		kind:     popupSecret,
		title:    "OAuth client secret (blank if public client)",
		subtitle: "Google desktop clients issue one; PKCE-only servers do not",
		input:    newPopupInput("", false),
		onConfirm: func(value string) any {
			return McpClientSecret{secret: strings.TrimSpace(value)}
		},
	}
	return t, nil
}

func (t TUI) openMcpClientRedirect() (TUI, tea.Cmd) {
	t.popup = &Popup{
		kind:     popupText,
		title:    "OAuth redirect URI (blank = " + mcp.DefaultRedirectURI + ")",
		subtitle: "must match the provider console exactly",
		input:    newPopupInput("", false),
		onConfirm: func(value string) any {
			return McpClientRedirect{uri: strings.TrimSpace(value)}
		},
	}
	return t, nil
}

func (t TUI) runMcpClientSave(redirectURI string) (TUI, tea.Cmd) {
	draft := t.mcpClient
	t.mcpClient = nil
	if draft == nil {
		return t, tea.Println(errorStyle.Render("[!] mcp client state lost") + "\n")
	}
	if err := mcp.ClearOAuth(draft.server); err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] mcp.ClearOAuth: %v", err)) + "\n")
	}
	if err := mcp.SaveOAuthClient(draft.server, draft.id, draft.secret, redirectURI); err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] mcp.SaveOAuthClient: %v", err)) + "\n")
	}
	return t.startMcpLogin(draft.server)
}

func (t TUI) startMcpLogin(name string) (TUI, tea.Cmd) {
	ctx, cancel := context.WithTimeout(t.ctx, 10*time.Minute)

	t.mcpOAuth = &oauthState{
		provider:  name,
		mcpServer: name,
		cancel:    cancel,
	}
	t.popup = &Popup{
		kind:     popupOAuth,
		title:    fmt.Sprintf("%s OAuth · discovering authorization server…", name),
		subtitle: "browser will open automatically once the URL is ready",
		oauth:    t.mcpOAuth,
	}

	go func() {
		err := mcp.Login(ctx, name, func(url string) {
			send(McpOAuthInfo{url: url})
		})
		cancel()
		send(McpOAuthDone{name: name, err: err})
	}()
	return t, nil
}

func (t TUI) openMcpOAuthPaste(state *oauthState) (tea.Model, tea.Cmd) {
	t.popup = &Popup{
		kind:     popupText,
		title:    fmt.Sprintf("%s OAuth · paste the redirect URL", state.mcpServer),
		subtitle: "for browsers that cannot reach this machine's loopback listener",
		input:    newPopupInput("", false),
		oauth:    state,
		onConfirm: func(value string) any {
			return McpOAuthPaste{server: state.mcpServer, url: strings.TrimSpace(value)}
		},
	}
	return t, nil
}

func (t TUI) runMcpOAuthPaste(msg McpOAuthPaste) (TUI, tea.Cmd) {
	state := t.mcpOAuth
	if state == nil {
		state = &oauthState{provider: msg.server, mcpServer: msg.server}
	}
	t.popup = &Popup{
		kind:     popupOAuth,
		title:    fmt.Sprintf("%s OAuth · waiting for authorization…", msg.server),
		subtitle: "",
		oauth:    state,
	}
	if err := mcp.SubmitCallback(msg.server, msg.url); err != nil {
		return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] %s oauth paste: %v", msg.server, err)) + "\n")
	}
	return t, nil
}

func (t TUI) runMcpOAuthInfo(msg McpOAuthInfo) (TUI, tea.Cmd) {
	if t.popup == nil || t.popup.kind != popupOAuth || t.popup.oauth == nil {
		return t, nil
	}
	t.popup.oauth.url = msg.url
	t.popup.title = fmt.Sprintf("%s OAuth · open browser to authorize", t.popup.oauth.provider)
	t.popup.subtitle = ""
	if msg.url != "" {
		openBrowser(msg.url)
	}
	return t, nil
}

func (t TUI) runMcpOAuthDone(msg McpOAuthDone) (TUI, tea.Cmd) {
	t.popup = nil
	t.mcpOAuth = nil
	switch {
	case msg.err == nil:
		next, cmd := t.reconnectMcpServer(msg.name)
		return next, tea.Batch(tea.Println(hintStyle.Render(fmt.Sprintf("⎯ %s · oauth authorized", msg.name))), cmd)
	case errors.Is(msg.err, context.Canceled):
		return t, tea.Println(hintStyle.Render("⎯ oauth cancelled") + "\n")
	case errors.Is(msg.err, context.DeadlineExceeded):
		return t, tea.Println(warnStyle.Render("⎯ oauth timed out") + "\n")
	}
	return t, tea.Println(errorStyle.Render(fmt.Sprintf("[!] %s oauth: %v", msg.name, msg.err)) + "\n")
}
