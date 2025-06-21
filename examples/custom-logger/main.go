package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/coder/websockify"
)

// slogAdapter adapts Go's structured logger to websockify.Logger interface
type slogAdapter struct {
	logger *slog.Logger
}

func (s *slogAdapter) Printf(format string, v ...interface{}) {
	s.logger.Info(fmt.Sprintf(format, v...))
}

func (s *slogAdapter) Println(v ...interface{}) {
	s.logger.Info(fmt.Sprint(v...))
}

func main() {
	// Create a structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Configure websockify with custom logger
	config := websockify.Config{
		Listener: ":8080",
		Target:   "localhost:5900",
		Logger:   &slogAdapter{logger: logger},
	}

	server := websockify.New(config)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger.Info("Starting websockify proxy with structured logging")

	if err := server.Serve(ctx); err != nil {
		logger.Error("Server failed", "error", err)
	}

	logger.Info("Server stopped")
}