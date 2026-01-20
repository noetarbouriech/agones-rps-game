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

	router.AddConsumerHandler(
		"matchmaking_handler", // Name of the handler
		"matchmaking",         // Topic to subscribe to
		s.sub,                 // Subscriber
		s.matchmakingHandler,  // Handler function
	)

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
	router.Close()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Failed to shutdown HTTP server: %v", err)
	}
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

	// Simulate a player joining matchmaking
	playerID := rand.Text()
	matchmakingTopic := "matchmaking"
	playerResultTopic := fmt.Sprintf("match_results_%s", playerID)

	// Publish the matchmaking request
	msg := message.NewMessage(watermill.NewUUID(), []byte(playerID))
	if err := s.pub.Publish(matchmakingTopic, msg); err != nil {
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
	for {
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
}

func (s *Server) matchmakingHandler(msg *message.Message) error {
	// Process the matchmaking message
	playerID := string(msg.Payload)
	log.Printf("Processing player: %s", playerID)

	// Simulate matchmaking logic (e.g., pairing players)
	matchResult := fmt.Sprintf("https://large-type.com/#%s", playerID)

	// Publish the match result to the player's specific topic
	playerResultTopic := fmt.Sprintf("match_results_%s", playerID)
	resultMsg := message.NewMessage(watermill.NewUUID(), []byte(matchResult))
	if err := s.pub.Publish(playerResultTopic, resultMsg); err != nil {
		log.Printf("Failed to publish match result: %v", err)
		return err
	}

	// Acknowledge the original message
	return nil
}
