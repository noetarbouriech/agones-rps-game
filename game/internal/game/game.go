package game

import (
	"log"
	"slices"
	"sync"
)

type Game struct {
	players []*Player
	mu      sync.Mutex
}

func NewGame() *Game {
	return &Game{
		players: make([]*Player, 0, 2),
	}
}

func (g *Game) AddPlayer(player *Player) {
	if len(g.players) >= 2 || player == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	g.players = append(g.players, player)
	log.Printf("Player %p added to game", player)
}

func (g *Game) RemovePlayer(player *Player) {
	if len(g.players) == 0 || player == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	idx := slices.Index(g.players, player)
	if idx != -1 {
		g.players = slices.Delete(g.players, idx, idx+1)
	}
	log.Printf("Player %p removed from game", player)
}

func (g *Game) PlayMove(player *Player, move string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	player.move = move
	log.Printf("Player %p played %s", player, move)
}

func (g *Game) Ended() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.players) == 2 && g.players[0].move != "" && g.players[1].move != ""
}

func (g *Game) SendResults() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.players) < 2 {
		return
	}

	p1, p2 := g.players[0], g.players[1]
	log.Printf("Player %p vs Player %p", p1, p2)

	p1Msg := "You: " + p1.move + "<br>Opponent: " + p2.move
	p2Msg := "You: " + p2.move + "<br>Opponent: " + p1.move

	winner := resolveWinner(p1, p2)
	log.Println("Winner:", winner)

	switch winner {
	case p1:
		p1.Send(p1Msg + "<br><b>You won 🎉</b>")
		p2.Send(p2Msg + "<br><b>You lost 😢</b>")
	case p2:
		p1.Send(p1Msg + "<br><b>You lost 😢</b>")
		p2.Send(p2Msg + "<br><b>You won 🎉</b>")
	default:
		p1.Send(p1Msg + "<br><b>Draw 🤝</b>")
		p2.Send(p2Msg + "<br><b>Draw 🤝</b>")
	}
}

func resolveWinner(p1, p2 *Player) *Player {
	// if tie or one player didn't make a move
	if p1.move == p2.move || p1.move == "" || p2.move == "" {
		return nil
	}

	if (p1.move == "rock" && p2.move == "scissors") ||
		(p1.move == "paper" && p2.move == "rock") ||
		(p1.move == "scissors" && p2.move == "paper") {
		return p1
	}
	return p2
}
