package main

import (
	"context"
	"crypto/rand"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/noetarbouriech/agones-rps-game/matchmaking/internal/matcher"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

//go:embed index.html
var index embed.FS

var (
	logger   = watermill.NewStdLogger(false, false)
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

type Server struct {
	pub    message.Publisher
	sub    message.Subscriber
	ctx    context.Context
	cancel context.CancelFunc
}

func main() {
	// Set up signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	pubSub := gochannel.NewGoChannel(gochannel.Config{}, logger)

	// Inititalize the server
	s := Server{
		pub:    pubSub,
		sub:    pubSub,
		ctx:    ctx,
		cancel: cancel,
	}

	// Start Watermill router
	router, _ := message.NewRouter(message.RouterConfig{}, logger)

	// Initialize matcher
	matcher := matcher.NewMatcher(s.pub, s.sub)
	go func() {
		log.Println("Starting Matcher")
		if err := matcher.Run(ctx); err != nil {
			log.Fatalf("Matcher router failed: %v", err)
		}
	}()

	go func() {
		if err := router.Run(ctx); err != nil {
			panic(err)
		}
	}()

	// Initialize HTTP server
	httpServer := s.newHTTPServer()
	go func() {
		log.Println("Starting HTTP server on port 3000")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	// Create a context with a timeout for graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutting down everything
	log.Println("Shutting down...")
	matcher.Shutdown()
	router.Close()
	httpServer.Shutdown(shutdownCtx)
}

func (s *Server) newHTTPServer() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(http.FS(index)).ServeHTTP(w, r)
	})
	mux.HandleFunc("/ws", s.ws)
	return &http.Server{
		Addr:         ":3000",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  0, // I don't want my websockets to timeout
	}
}

func (s *Server) ws(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	defer conn.Close() // Ensure connection is always closed when the handler exits.

	playerID := rand.Text() // random player ID
	playerResultTopic := fmt.Sprintf("match_results_%s", playerID)

	// Publish the matchmaking request
	msg := message.NewMessage(watermill.NewUUID(), []byte(playerID))
	if err := s.pub.Publish("matchmaking", msg); err != nil {
		log.Printf("Failed to publish matchmaking message: %v", err)
		return
	}

	// Subscribe to the player's result topic
	messages, err := s.sub.Subscribe(s.ctx, playerResultTopic)
	if err != nil {
		log.Printf("Failed to subscribe to player result topic: %v", err)
		return
	}

	// Wait for a match result
	select {
	case <-s.ctx.Done():
		return // Exit if the server is shutting down
	case msg := <-messages:
		matchResult := string(msg.Payload)
		log.Printf("Match found for player %s: %s", playerID, matchResult)

		// Send the match result back to the WebSocket client
		if err := conn.WriteMessage(websocket.TextMessage, []byte(matchResult)); err != nil {
			log.Printf("Failed to send match result: %v", err)
			return
		}

		// Acknowledge the message and exit
		msg.Ack()
		return
	}
}
