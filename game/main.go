package main

import (
	"context"
	"embed"
	"log"
	"os/signal"
	"syscall"
	"time"

	"net/http"

	sdk "agones.dev/agones/sdks/go"
	"github.com/gorilla/websocket"
	"github.com/noetarbouriech/agones-rps-game/game/internal/agones"
	"github.com/noetarbouriech/agones-rps-game/game/internal/game"
)

//go:embed index.html
var index embed.FS

type Server struct {
	game   *game.Game
	sdk    *sdk.SDK
	ctx    context.Context
	cancel context.CancelFunc
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	// Initialize Agones SDK
	log.Println("Creating SDK instance")
	sdk, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	// Set up signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// Inititalize the server
	s := Server{
		game:   game.NewGame(),
		sdk:    sdk,
		ctx:    ctx,
		cancel: cancel,
	}

	// Mark server ready
	if err := sdk.Ready(); err != nil {
		log.Fatalf("Failed to mark Ready: %v", err)
	}

	// Health Ping for Agones SDK
	log.Println("Starting Health Ping")
	go agones.HealthPing(s.sdk, s.ctx)

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
	s.sdk.Shutdown()
	s.game.Shutdown()
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

	if s.game.Ended() == true {
		return
	}

	player := game.NewPlayer(conn)
	s.game.AddPlayer(player)

	// Read the first message which should contain the player's move
	msgType, msg, err := conn.ReadMessage()
	if err != nil {
		// Client disconnected or error occurred immediately
		log.Printf("Player %p left", player)
		s.game.RemovePlayer(player)
		return
	}

	// Check is message is valid
	if msgType != websocket.TextMessage {
		log.Printf("Player %p message is invalid.", player)
		return
	}

	s.game.PlayMove(player, string(msg))

	// Check if the two players have played
	if s.game.Ended() {
		s.game.SendResults()
		time.Sleep(5 * time.Second)
		s.game.Shutdown()
		s.cancel() // Cancel the context to stop the game server
		return
	}

	// First player gets into a loop waiting for the opponent's  move
	player.Send("Waiting for opponent...\n")
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			// Client disconnected or error occurred immediately
			log.Printf("Player %p left", player)
			s.game.RemovePlayer(player)
			return
		}
	}
}
