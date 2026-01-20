package game

import (
	"errors"
	"log"

	"github.com/gorilla/websocket"
)

type Player struct {
	conn *websocket.Conn
	move string
}

func NewPlayer(conn *websocket.Conn) *Player {
	return &Player{
		conn: conn,
	}
}

func (p *Player) Send(msg string) error {
	log.Printf("Sending message to %p: %s", p, msg)
	if p == nil || p.conn == nil {
		return errors.New("Failed to send message")
	}
	return p.conn.WriteMessage(websocket.TextMessage, []byte(msg))
}
