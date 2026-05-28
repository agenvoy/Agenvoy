package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/session"
	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func registRegistryEmail() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "get_registry_email",
		AlwaysAllow: true,
		Description: "Read the marketplace registry email from ~/.config/agenvoy/config.json. Returns {email: \"\"} when not set.",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Handler: func(_ context.Context, _ *toolTypes.Executor, _ json.RawMessage) (string, error) {
			cfg, err := session.Load()
			if err != nil {
				return "", fmt.Errorf("session.Load: %w", err)
			}
			out, err := json.Marshal(map[string]any{"email": cfg.RegistryEmail})
			if err != nil {
				return "", fmt.Errorf("json.Marshal: %w", err)
			}
			return string(out), nil
		},
	})

	toolRegister.Regist(toolRegister.Def{
		Name:        "set_registry_email",
		AlwaysAllow: true,
		Description: "Save the marketplace registry email to ~/.config/agenvoy/config.json. Validates strict email format.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"email": map[string]any{
					"type":        "string",
					"description": "Email address. Must match ^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$.",
				},
			},
			"required": []string{"email"},
		},
		Handler: func(_ context.Context, _ *toolTypes.Executor, args json.RawMessage) (string, error) {
			var p struct {
				Email string `json:"email"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			email := strings.TrimSpace(p.Email)
			if email == "" {
				return "", fmt.Errorf("email is required")
			}
			if !emailPattern.MatchString(email) {
				return "", fmt.Errorf("invalid email format: %q", email)
			}

			cfg, err := session.Load()
			if err != nil {
				return "", fmt.Errorf("session.Load: %w", err)
			}
			cfg.RegistryEmail = email
			if err := session.Save(cfg); err != nil {
				return "", fmt.Errorf("session.Save: %w", err)
			}

			out, err := json.Marshal(map[string]any{"ok": true, "email": email})
			if err != nil {
				return "", fmt.Errorf("json.Marshal: %w", err)
			}
			return string(out), nil
		},
	})
}
