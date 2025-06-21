package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coder/websockify"
)

func main() {
	// Create websockify proxy (no built-in HTTP server)
	config := websockify.Config{
		Target: "localhost:5900", // Only need target, no listener
		Logger: &websockify.NoOpLogger{}, // Use our own logging
	}

	proxy := websockify.New(config)

	// Create HTTP server with multiple endpoints
	mux := http.NewServeMux()
	
	// Mount websockify at /vnc endpoint
	mux.Handle("/vnc", proxy)
	
	// Add other endpoints
	mux.HandleFunc("/health", healthCheck)
	mux.HandleFunc("/api/status", statusAPI)
	mux.HandleFunc("/", homeHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start HTTP server
	go func() {
		log.Println("Starting HTTP server on :8080")
		log.Println("WebSocket endpoint: ws://localhost:8080/vnc")
		log.Println("Health check: http://localhost:8080/health")
		log.Println("Status API: http://localhost:8080/api/status")
		log.Println("Home page: http://localhost:8080/")
		
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Shutting down...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func statusAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":  "websockify-proxy",
		"version":  "1.0.0",
		"endpoint": "/vnc",
		"target":   "localhost:5900",
		"uptime":   time.Since(startTime).String(),
	})
}

var startTime = time.Now()

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title>WebSockify Proxy</title>
</head>
<body>
    <h1>WebSockify Proxy Server</h1>
    <p>This server provides WebSocket to TCP proxy functionality.</p>
    <ul>
        <li><strong>WebSocket Endpoint:</strong> <code>ws://localhost:8080/vnc</code></li>
        <li><strong>Target:</strong> <code>localhost:5900</code></li>
        <li><a href="/health">Health Check</a></li>
        <li><a href="/api/status">Status API</a></li>
    </ul>
</body>
</html>`))
}