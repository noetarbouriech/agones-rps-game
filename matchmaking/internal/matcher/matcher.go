package matcher

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Player string

type Matcher struct {
	pub     message.Publisher
	sub     message.Subscriber
	router  *message.Router
	waiting Player
	mu      *sync.Mutex
}

func NewMatcher(pub message.Publisher, sub message.Subscriber) *Matcher {
	logger := watermill.NewStdLogger(false, false)

	router, _ := message.NewRouter(message.RouterConfig{}, logger)

	m := &Matcher{
		pub:     pub,
		sub:     sub,
		router:  router,
		waiting: "",
		mu:      &sync.Mutex{},
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

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.waiting == "" {
		m.waiting = Player(playerID)
		return nil
	}

	var matchResult string
	var err error
	retryInterval := 5 * time.Second

	for {
		matchResult, err = AllocateGameServer()
		if err == nil {
			break
		}
		log.Printf("Failed to allocate game server: %v", err)
		time.Sleep(retryInterval)
	}

	resultMsg := message.NewMessage(watermill.NewUUID(), []byte(matchResult))

	// Publish the match result to the player's topic
	playerResultTopic := fmt.Sprintf("match_results_%s", playerID)
	if err := m.pub.Publish(playerResultTopic, resultMsg); err != nil {
		log.Printf("Failed to publish match result: %v", err)
		return err
	}

	// Publish the match result to the waiting player's topic
	waitingResultTopic := fmt.Sprintf("match_results_%s", m.waiting)
	if err := m.pub.Publish(waitingResultTopic, resultMsg); err != nil {
		log.Printf("Failed to publish match result: %v", err)
		return err
	}

	// remove waiting player
	m.waiting = ""

	// no error
	return nil
}

func (m *Matcher) Shutdown() error {
	return m.router.Close()
}
