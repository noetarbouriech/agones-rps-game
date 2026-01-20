package matcher

import (
	"context"
	"fmt"
	"log"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Matcher struct {
	pub    message.Publisher
	sub    message.Subscriber
	router *message.Router
}

func NewMatcher(pub message.Publisher, sub message.Subscriber) *Matcher {
	logger := watermill.NewStdLogger(false, false)

	router, _ := message.NewRouter(message.RouterConfig{}, logger)

	m := &Matcher{
		pub:    pub,
		sub:    sub,
		router: router,
	}

	m.router.AddConsumerHandler(
		"matchmaking_handler", // Name of the handler
		"matchmaking",         // Topic to subscribe to
		m.sub,                 // Subscriber
		m.matchmakingHandler,  // Handler function,
	)
	return m
}

func (m *Matcher) Run(ctx context.Context) error {
	return m.router.Run(ctx)
}

func (m *Matcher) matchmakingHandler(msg *message.Message) error {
	// Process the matchmaking message
	playerID := string(msg.Payload)
	log.Printf("Processing player: %s", playerID)

	// Simulate matchmaking logic (e.g., pairing players)
	matchResult := fmt.Sprintf("https://large-type.com/#%s", playerID)

	// Publish the match result to the player's specific topic
	playerResultTopic := fmt.Sprintf("match_results_%s", playerID)
	resultMsg := message.NewMessage(watermill.NewUUID(), []byte(matchResult))
	if err := m.pub.Publish(playerResultTopic, resultMsg); err != nil {
		log.Printf("Failed to publish match result: %v", err)
		return err
	}

	// Acknowledge the original message
	return nil
}

func (m *Matcher) Shutdown() error {
	return m.router.Close()
}
