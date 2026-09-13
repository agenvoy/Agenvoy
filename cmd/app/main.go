package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	osexec "os/exec"
	"os/signal"
	"syscall"

	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime"
	"github.com/pardnchiu/agenvoy/internal/runtime/mcp"
	go_pkg_sandbox "github.com/pardnchiu/go-pkg/sandbox"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "stop":
			stop()
			return

		case "update":
			update()
			return

		case "--daemon":
			Daemon()
			return

		default:
			usage()
			os.Exit(1)
		}
	}

	if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
		mcpServer()
		return
	}

	TUI()
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  agen                                            Attach TUI; spawn server daemon if not running")
	fmt.Println("  agen stop                                       Stop the running server daemon")
	fmt.Println("  agen update                                     Update agen to the latest release")
}

func stop() {
	if err := filesystem.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "filesystem.Init: %v\n", err)
		os.Exit(1)
	}
	r, err := runtime.Read()
	if err != nil || r == nil {
		fmt.Println("No daemon running.")
		return
	}
	if !runtime.IsAlive(r.PID) {
		fmt.Printf("Daemon record stale (pid=%d not alive); clearing.\n", r.PID)
		_ = runtime.Clear()
		return
	}
	fmt.Printf("Stopping daemon (pid=%d)...\n", r.PID)
	if err := runtime.Stop(r.PID); err != nil {
		fmt.Fprintf(os.Stderr, "runtime.Stop: %v\n", err)
		os.Exit(1)
	}
	if err := runtime.Clear(); err != nil {
		slog.Warn("runtime.Clear",
			slog.String("error", err.Error()))
	}
	fmt.Println("Daemon stopped.")
}

func update() {
	const remoteURL = "https://agenvoy.com/scripts/update.sh"

	f, err := os.CreateTemp("", "agenvoy-update-*.sh")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp: %v\n", err)
		os.Exit(1)
	}
	tmpPath := f.Name()
	f.Close()

	cleanup := func() { _ = os.Remove(tmpPath) }
	defer cleanup()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cleanup()
		os.Exit(130)
	}()

	fmt.Printf("Fetching updater from %s -> %s\n", remoteURL, tmpPath)
	curl := osexec.Command("curl", "-fsSL", remoteURL, "-o", tmpPath)
	curl.Stdout = os.Stdout
	curl.Stderr = os.Stderr
	if err := curl.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "download failed: %v\n", err)
		os.Exit(1)
	}

	cmd := osexec.Command("bash", tmpPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "update failed: %v\n", err)
		os.Exit(1)
	}
}

func mcpServer() {
	if err := filesystem.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "filesystem.Init: %v\n", err)
		os.Exit(1)
	}
	if err := filesystem.LoadRuntime(); err != nil {
		slog.Warn("filesystem.LoadRuntime",
			slog.String("error", err.Error()))
	}
	if err := go_pkg_sandbox.CheckDependence(); err != nil {
		slog.Warn("sandbox.CheckDependence",
			slog.String("error", err.Error()))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	server := mcp.NewServer()
	if err := server.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "mcpserver: %v\n", err)
		os.Exit(1)
	}
}
