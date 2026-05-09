package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pardnchiu/agenvoy/internal/interactive/discord"
	"github.com/pardnchiu/agenvoy/internal/session"
	"github.com/pardnchiu/go-pkg/filesystem/keychain"
	"golang.org/x/term"
)

func runDiscord(args []string) {
	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(strings.TrimSpace(args[0]))
	}
	switch sub {
	case "enable":
		runDiscordEnable()

	case "disable":
		runDiscordDisable()

	default:
		fmt.Fprintf(os.Stderr, "Usage: agen discord [enable|disable]\n")
		os.Exit(1)
	}
}

func runDiscordEnable() {
	cfg, err := session.Load()
	if err != nil {
		slog.Error("session.Load", slog.String("error", err.Error()))
		os.Exit(1)
	}

	keepToken := false
	if existing := keychain.Get(discord.Key); existing != "" {
		confirm := promptui.Select{
			Label:        "Discord token exists, replace?",
			Items:        []string{"No, use existing", "Yes, replace"},
			HideSelected: true,
		}
		idx, _, err := confirm.Run()
		if err != nil {
			os.Exit(1)
		}
		keepToken = idx == 0
	}

	if !keepToken {
		fmt.Print("Discord Bot Token: ")
		keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			slog.Error("term.ReadPassword", slog.String("error", err.Error()))
			os.Exit(1)
		}
		defer func() {
			for i := range keyBytes {
				keyBytes[i] = 0
			}
		}()
		token := strings.TrimSpace(string(keyBytes))
		if token == "" {
			slog.Error("token is required")
			os.Exit(1)
		}
		if err := keychain.Set(discord.Key, token); err != nil {
			slog.Error("keychain.Set",
				slog.String("error", err.Error()))
			os.Exit(1)
		}
		fmt.Printf("[*] %s saved\n", discord.Key)
	}

	guildPrompt := promptui.Prompt{
		Label: "Guild ID (leave empty for global slash commands)",
	}
	rawGuild, err := guildPrompt.Run()
	if err != nil {
		os.Exit(1)
	}

	cfg.DiscordGuildID = strings.TrimSpace(rawGuild)
	cfg.DiscordEnabled = true

	if err := session.Save(cfg); err != nil {
		slog.Error("session.Save", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func runDiscordDisable() {
	cfg, err := session.Load()
	if err != nil {
		slog.Error("session.Load", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if !cfg.DiscordEnabled && keychain.Get(discord.Key) == "" {
		fmt.Println("[*] discord already disabled")
		return
	}
	if err := keychain.Delete(discord.Key); err != nil {
		slog.Warn("keychain.Delete",
			slog.String("error", err.Error()))
	}

	cfg.DiscordEnabled = false
	if err := session.Save(cfg); err != nil {
		slog.Error("session.Save", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
