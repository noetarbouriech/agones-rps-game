package main

import (
	"context"
	"embed"
	"log"

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Inititalize the server
	s := Server{
		game:   game.NewGame(),
		sdk:    sdk,
		ctx:    ctx,
		cancel: cancel,
	}

	// Health Ping for Agones SDK
	log.Println("Starting Health Ping")
	go agones.HealthPing(s.sdk, s.ctx)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(http.FS(index)).ServeHTTP(w, r)
	})
	http.HandleFunc("/ws", s.ws)
	log.Println("Starting HTTP server on port 3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func (s *Server) ws(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	defer conn.Close() // Ensure connection is always closed when the handler exits.

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
