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

// Server represents a websockify server that can proxy websocket connections to TCP targets.
type Server struct {
	listener string
	target   string
	webRoot  string
	server   *http.Server
}

// Config holds the configuration for the websockify server.
type Config struct {
	Listener string
	Target   string
	WebRoot  string
}

// New creates a new websockify server with the given configuration.
func New(config Config) *Server {
	return &Server{
		listener: config.Listener,
		target:   config.Target,
		webRoot:  config.WebRoot,
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
		log.Println("Refusing to serve static content from the current working directory.")
		log.Println("Please use the --web-root flag to specify a different directory.")
		log.Println("Exiting.")
		return nil
	case s.webRoot == "":
		log.Println("No web root specified; serving no static content.")
	default:
		log.Printf("Serving %s at %s", s.webRoot, s.listener)
		mux.Handle("/", http.FileServer(http.Dir(s.webRoot)))
	}

	log.Printf("Serving WS of %s at %s", s.target, s.listener)
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

func (s *Server) forwardTCP(wsConn *websocket.Conn, conn net.Conn) {
	var tcpBuffer [1024]byte
	defer func() {
		if conn != nil {
			conn.Close()
		}
		if wsConn != nil {
			wsConn.Close()
		}
	}()
	for {
		if (conn == nil) || (wsConn == nil) {
			return
		}
		n, err := conn.Read(tcpBuffer[0:])
		if err != nil {
			log.Printf("reading from TCP failed: %s", err)
			return
		}

		if err := wsConn.WriteMessage(websocket.BinaryMessage, tcpBuffer[0:n]); err != nil {
			log.Printf("writing to WS failed: %s", err)
		}
	}
}

func (s *Server) forwardWeb(wsConn *websocket.Conn, conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("reading from WS failed: %s", err)
		}
		if conn != nil {
			conn.Close()
		}
		if wsConn != nil {
			wsConn.Close()
		}
	}()
	for {
		if (conn == nil) || (wsConn == nil) {
			return
		}

		_, buffer, err := wsConn.ReadMessage()
		if err == nil {
			if _, err := conn.Write(buffer); err != nil {
				log.Printf("writing to TCP failed: %s", err)
			}
		}
	}
}

func (s *Server) newServeWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("failed to upgrade to WS: %s", err)
			return
		}

		vnc, err := net.Dial("tcp", s.target)
		if err != nil {
			log.Printf("failed to bind to the target: %s", err)
		}

		go s.forwardTCP(ws, vnc)
		go s.forwardWeb(ws, vnc)
	}
}
