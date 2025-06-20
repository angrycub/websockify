package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	var (
		port = flag.String("port", "5901", "Port to listen on")
		help = flag.Bool("help", false, "Show this help message")
	)
	flag.Parse()

	if *help {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "echoserver - Simple TCP echo server for testing websockify\n\n")
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -port 5901\n", os.Args[0])
		os.Exit(0)
	}

	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", *port, err)
	}
	defer listener.Close()

	log.Printf("Echo server listening on port %s", *port)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down echo server...")
		listener.Close()
		os.Exit(0)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			// Check if the error is due to the listener being closed
			if strings.Contains(err.Error(), "use of closed network connection") {
				log.Println("Listener closed, stopping accept loop")
				return
			}
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleEchoConnection(conn)
	}
}

func handleEchoConnection(conn net.Conn) {
	defer conn.Close()
	
	clientAddr := conn.RemoteAddr().String()
	log.Printf("New echo connection from %s", clientAddr)

	// Simple echo: copy everything from conn back to conn
	_, err := io.Copy(conn, conn)
	if err != nil {
		log.Printf("Echo connection from %s ended: %v", clientAddr, err)
	} else {
		log.Printf("Echo connection from %s closed", clientAddr)
	}
}