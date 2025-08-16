package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/openshift/rosa-hcp/cmd/rosa/commands"
	"github.com/openshift/rosa-hcp/internal/config"
	"github.com/openshift/rosa-hcp/internal/version"
)

func main() {
	// Set up structured logging
	logger := setupLogger()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("received interrupt signal, shutting down...")
		cancel()
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Warn("failed to load config file, using defaults",
			slog.String("error", err.Error()))
		cfg = config.Default()
	}

	// Create and execute root command
	rootCmd := commands.NewRootCommand(ctx, cfg, logger)
	rootCmd.Version = version.Version

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		logger.Error("command execution failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func setupLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	// Check for debug mode
	if os.Getenv("ROSA_DEBUG") == "true" {
		opts.Level = slog.LevelDebug
	}

	// Use JSON logging if not a terminal
	if os.Getenv("ROSA_LOG_FORMAT") == "json" {
		handler := slog.NewJSONHandler(os.Stderr, opts)
		return slog.New(handler)
	}

	// Default to text handler for terminal
	handler := slog.NewTextHandler(os.Stderr, opts)
	return slog.New(handler)
}
