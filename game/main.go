package main

import (
	"context"
	"embed"
	"log"
	"time"

	"net/http"

	sdk "agones.dev/agones/sdks/go"
	"github.com/gorilla/websocket"
)

//go:embed index.html
var index embed.FS

var game *Game

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	// Initialize Agones SDK
	log.Println("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	// Health Ping for Agones SDK
	log.Println("Starting Health Ping")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go doHealth(s, ctx)

	// Inititalize the game
	game = NewGame()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(http.FS(index)).ServeHTTP(w, r)
	})
	http.HandleFunc("/ws", game.ws)
	log.Println("Starting HTTP server on port 3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func (game *Game) ws(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	defer conn.Close() // Ensure connection is always closed when the handler exits.

	player := &Player{conn: conn}
	game.AddPlayer(player)

	// Read the first message which should contain the player's move
	msgType, msg, err := conn.ReadMessage()
	if err != nil {
		// Client disconnected or error occurred immediately
		log.Printf("Player %p left", player)
		game.RemovePlayer(player)
		return
	}

	// Check is message is valid
	if msgType != websocket.TextMessage {
		log.Printf("Player %p is invalid.", player)
		return
	}

	game.PlayMove(player, string(msg))

	// Check if the two players have played
	if game.Ended() {
		game.SendResults()
		return
	}

	// First player gets into a loop waiting for the opponent's  move
	player.Send("Waiting for opponent...\n")
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			// Client disconnected or error occurred immediately
			log.Printf("Player %p left", player)
			game.RemovePlayer(player)
			return
		}
	}

}

// doHealth sends the regular Health Pings to Agones SDK
func doHealth(sdk *sdk.SDK, ctx context.Context) {
	tick := time.Tick(2 * time.Second)
	for {
		err := sdk.Health()
		if err != nil {
			log.Fatalf("Could not send health ping, %v", err)
		}
		select {
		case <-ctx.Done():
			log.Print("Stopped health pings")
			return
		case <-tick:
		}
	}
}
