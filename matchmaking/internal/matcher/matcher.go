package matcher

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"agones.dev/agones/pkg/client/clientset/versioned"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Player string

type Matcher struct {
	pub          message.Publisher
	sub          message.Subscriber
	router       *message.Router
	waiting      Player
	mu           *sync.Mutex
	agonesClient *versioned.Clientset
}

func NewMatcher(pub message.Publisher, sub message.Subscriber) *Matcher {
	logger := watermill.NewStdLogger(false, false)

	router, _ := message.NewRouter(message.RouterConfig{}, logger)

	agonesClient, err := NewAgonesClient()
	if err != nil {
		log.Fatalf("Could not create Agones clientset: %v", err)
	}

	m := &Matcher{
		pub:          pub,
		sub:          sub,
		router:       router,
		waiting:      "",
		mu:           &sync.Mutex{},
		agonesClient: agonesClient,
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
	if m.waiting == "" {
		m.waiting = Player(playerID)
		m.mu.Unlock()
		return nil
	}

	waitingPlayer := m.waiting // copy waiting player to unlock mu
	m.waiting = ""
	m.mu.Unlock()

	var matchResult string
	var err error
	retryInterval := 5 * time.Second

	for {
		matchResult, err = m.AllocateGameServer(msg.Context())
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
	waitingResultTopic := fmt.Sprintf("match_results_%s", waitingPlayer)
	if err := m.pub.Publish(waitingResultTopic, resultMsg); err != nil {
		log.Printf("Failed to publish match result: %v", err)
		return err
	}

	// no error
	return nil
}

func (m *Matcher) Shutdown() error {
	return m.router.Close()
}
