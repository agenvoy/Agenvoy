package auth

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	verifyTimeout = 20 * time.Second
)

func Cached(ctx context.Context) bool {
	if os.Geteuid() == 0 {
		return true
	}
	probeCtx, cancel := context.WithTimeout(ctx, verifyTimeout)
	defer cancel()
	return exec.CommandContext(probeCtx, "sudo", "-n", "-v").Run() == nil
}

func Verify(ctx context.Context, password string) error {
	if os.Geteuid() == 0 {
		return nil
	}
	if Cached(ctx) {
		return nil
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}

	verifyCtx, cancel := context.WithTimeout(ctx, verifyTimeout)
	defer cancel()

	cmd := exec.CommandContext(verifyCtx, "sudo", "-S", "-p", "", "-v")
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
