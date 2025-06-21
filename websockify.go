package websockify

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

// Logger interface for custom logging implementations.
type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

// Server represents a websockify server that can proxy websocket connections to TCP targets.
type Server struct {
	listener string
	target   string
	webRoot  string
	server   *http.Server
	logger   Logger
}

// Config holds the configuration for the websockify server.
type Config struct {
	Listener string
	Target   string
	WebRoot  string
	Logger   Logger // Optional custom logger, defaults to standard log package
}

// defaultLogger wraps the standard log package to implement our Logger interface.
type defaultLogger struct{}

func (d *defaultLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (d *defaultLogger) Println(v ...interface{}) {
	log.Println(v...)
}

// NoOpLogger discards all log messages.
type NoOpLogger struct{}

func (n *NoOpLogger) Printf(format string, v ...interface{}) {}
func (n *NoOpLogger) Println(v ...interface{})               {}

// New creates a new websockify server with the given configuration.
func New(config Config) *Server {
	logger := config.Logger
	if logger == nil {
		logger = &defaultLogger{}
	}
	
	return &Server{
		listener: config.Listener,
		target:   config.Target,
		webRoot:  config.WebRoot,
		logger:   logger,
	}
}

// Serve starts the websockify server and blocks until the context is cancelled.
func (s *Server) Serve(ctx context.Context) error {
	path, err := os.Getwd()
	if err != nil {
		return err
	}

	mux := http.NewServeMux()

	switch {
	case s.webRoot == path:
		s.logger.Println("Refusing to serve static content from the current working directory.")
		s.logger.Println("Please use the --web-root flag to specify a different directory.")
		s.logger.Println("Exiting.")
		return nil
	case s.webRoot == "":
		s.logger.Println("No web root specified; serving no static content.")
	default:
		s.logger.Printf("Serving %s at %s", s.webRoot, s.listener)
		mux.Handle("/", http.FileServer(http.Dir(s.webRoot)))
	}

	s.logger.Printf("Serving WS of %s at %s", s.target, s.listener)
	mux.HandleFunc("/websockify", s.newServeWS())

	s.server = &http.Server{
		Addr:           s.listener,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		if s.server != nil {
			s.server.Close()
		}
	}()

	return s.server.ListenAndServe()
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") != ""
	},
}

// handleConnection manages the bidirectional forwarding for a single connection pair.
func (s *Server) handleConnection(ctx context.Context, wsConn *websocket.Conn, tcpConn net.Conn) {
	// Create a cancellable context for this connection
	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Clean up connections when done
	defer func() {
		if tcpConn != nil {
			tcpConn.Close()
		}
		if wsConn != nil {
			wsConn.Close()
		}
	}()

	// Channel to signal when either direction fails
	done := make(chan struct{}, 2)

	// Forward TCP -> WebSocket
	go s.forwardTCP(connCtx, wsConn, tcpConn, done)
	
	// Forward WebSocket -> TCP  
	go s.forwardWeb(connCtx, wsConn, tcpConn, done)

	// Wait for context cancellation or either goroutine to finish
	select {
	case <-connCtx.Done():
		s.logger.Printf("connection cancelled: %v", connCtx.Err())
	case <-done:
		// One direction failed, which will close connections and cause the other to fail
	}
}

func (s *Server) forwardTCP(ctx context.Context, wsConn *websocket.Conn, tcpConn net.Conn, done chan<- struct{}) {
	defer func() {
		select {
		case done <- struct{}{}:
		default:
		}
	}()

	var tcpBuffer [1024]byte
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Set read deadline to make read cancellable
		tcpConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		
		n, err := tcpConn.Read(tcpBuffer[0:])
		if err != nil {
			// Check if it's just a timeout, continue if context not cancelled
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			s.logger.Printf("reading from TCP failed: %s", err)
			return
		}

		if err := wsConn.WriteMessage(websocket.BinaryMessage, tcpBuffer[0:n]); err != nil {
			s.logger.Printf("writing to WS failed: %s", err)
			return
		}
	}
}

func (s *Server) forwardWeb(ctx context.Context, wsConn *websocket.Conn, tcpConn net.Conn, done chan<- struct{}) {
	defer func() {
		if err := recover(); err != nil {
			s.logger.Printf("WebSocket forwarding panic: %s", err)
		}
		select {
		case done <- struct{}{}:
		default:
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Set read deadline to make read cancellable
		wsConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		
		_, buffer, err := wsConn.ReadMessage()
		if err != nil {
			// Check if it's just a timeout, continue if context not cancelled
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Printf("WebSocket closed: %s", err)
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			s.logger.Printf("reading from WS failed: %s", err)
			return
		}

		if _, err := tcpConn.Write(buffer); err != nil {
			s.logger.Printf("writing to TCP failed: %s", err)
			return
		}
	}
}

// ServeHTTP implements http.Handler for integration with existing HTTP servers.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Printf("failed to upgrade to WS: %s", err)
		return
	}

	vnc, err := net.Dial("tcp", s.target)
	if err != nil {
		s.logger.Printf("failed to bind to the target: %s", err)
		if ws != nil {
			ws.Close()
		}
		return
	}

	// Use request context for connection lifecycle
	ctx := r.Context()
	s.handleConnection(ctx, ws, vnc)
}

func (s *Server) newServeWS() http.HandlerFunc {
	return s.ServeHTTP
}
