package agones

import (
	"context"
	"log"
	"time"

	sdk "agones.dev/agones/sdks/go"
)

// HealthPing sends regular health pings to the Agones SDK
func HealthPing(sdk *sdk.SDK, ctx context.Context) {
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
