package websocket

import (
	"context"
	"time"

	"github.com/SARVESHVARADKAR123/RealChat/services/delivery/internal/presence"
)

func StartHeartbeat(p *presence.Presence, userID, deviceID string, done <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		ctx := context.Background()

		for {
			select {
			case <-ticker.C:
				_ = p.Refresh(ctx, userID, deviceID)
			case <-done:
				return
			}
		}
	}()
}
