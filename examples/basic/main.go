package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/coder/websockify"
)

func main() {
	// Configure the websockify proxy
	config := websockify.Config{
		Listener: ":8080",                // Listen on port 8080
		Target:   "localhost:5900",       // Proxy to VNC server on port 5900
		WebRoot:  "",                     // No static files
	}

	// Create the server
	server := websockify.New(config)

	// Create context that cancels on interrupt signals
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Println("Starting websockify proxy...")
	log.Println("WebSocket endpoint: ws://localhost:8080/websockify")
	log.Println("Proxying to: localhost:5900")
	log.Println("Press Ctrl+C to stop")

	// Start server - blocks until context is cancelled
	if err := server.Serve(ctx); err != nil {
		log.Printf("Server error: %v", err)
	}

	log.Println("Server stopped")
}