package auth

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	verifyTimeout = 20 * time.Second
)

func Verify(ctx context.Context, password string) error {
	if password == "" {
		return fmt.Errorf("password is required")
	}

	verifyCtx, cancel := context.WithTimeout(ctx, verifyTimeout)
	defer cancel()

	cmd := exec.CommandContext(verifyCtx, "sudo", "-k", "-S", "-p", "", "-v")
	cmd.Stdin = strings.NewReader(password + "\n")
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if verifyCtx.Err() != nil {
		return fmt.Errorf("password check timed out")
	}
	if strings.Contains(strings.ToLower(string(out)), "incorrect password") {
		return fmt.Errorf("incorrect password")
	}
	return fmt.Errorf("password check failed")
}
