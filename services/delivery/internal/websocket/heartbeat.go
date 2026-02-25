package websocket

import (
	"context"
	"time"

	presencev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/presence/v1"
)

func StartHeartbeat(pc presencev1.PresenceApiClient, userID, deviceID string, done <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		ctx := context.Background()

		for {
			select {
			case <-ticker.C:
				_, _ = pc.RefreshSession(ctx, &presencev1.RefreshSessionRequest{
					UserId:   userID,
					DeviceId: deviceID,
				})
			case <-done:
				return
			}
		}
	}()
}
