package main

import (
	"context"
	"log"
	"time"

	sdk "agones.dev/agones/sdks/go"
)

func main() {
	// Initialize Agones SDK
	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	// Health Ping for Agones SDK
	log.Print("Starting Health Ping")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go doHealth(s, ctx)

	for {
		log.Println("Game running")
		time.Sleep(time.Second)
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
