package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coder/websockify"
)

func main() {
	// Configure websockify with no logging
	config := websockify.Config{
		Listener: ":8080",
		Target:   "localhost:5900",
		Logger:   &websockify.NoOpLogger{}, // Silent operation
	}

	server := websockify.New(config)

	// Create context with timeout (run for 30 seconds)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Also handle interrupt signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\nReceived interrupt signal, stopping...")
		cancel()
	}()

	fmt.Println("Starting silent websockify proxy...")
	fmt.Println("WebSocket endpoint: ws://localhost:8080/websockify")
	fmt.Println("Will run for 30 seconds or until Ctrl+C")

	start := time.Now()

	// Start server - runs silently
	if err := server.Serve(ctx); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

	duration := time.Since(start)
	fmt.Printf("Server ran for %v\n", duration.Round(time.Second))
}