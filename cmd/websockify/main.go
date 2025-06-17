package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/coder/websockify"
)

func main() {
	var (
		listener = flag.String("listen", "0.0.0.0:6080", "Host:port to listen on")
		target   = flag.String("target", "localhost:5900", "Host:port to connect to")
		webRoot  = flag.String("web-root", "", "Path to web files (leave empty for no static files)")
		help     = flag.Bool("help", false, "Show this help message")
	)
	flag.Parse()

	if *help {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "websockify - WebSocket to TCP proxy\n\n")
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -listen :8080 -target localhost:5900 -web-root ./web\n", os.Args[0])
		os.Exit(0)
	}

	config := websockify.Config{
		Listener: *listener,
		Target:   *target,
		WebRoot:  *webRoot,
	}

	server := websockify.New(config)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	log.Printf("Starting websockify server...")
	log.Printf("Listening on: %s", *listener)
	log.Printf("Proxying to: %s", *target)
	if *webRoot != "" {
		log.Printf("Web root: %s", *webRoot)
	}

	if err := server.Serve(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
