package presence

import (
	"context"

	presencev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/presence/v1"
)

type GRPCHandler struct {
	presencev1.UnimplementedPresenceApiServer
	presence *Presence
}

func NewGRPCHandler(p *Presence) *GRPCHandler {
	return &GRPCHandler{
		presence: p,
	}
}

func (h *GRPCHandler) GetPresence(ctx context.Context, req *presencev1.GetPresenceRequest) (*presencev1.GetPresenceResponse, error) {
	resp := &presencev1.GetPresenceResponse{}

	for _, userID := range req.UserIds {
		devices, err := h.presence.GetUserDevices(ctx, userID)
		if err != nil {
			// Log error but continue for other users
			continue
		}

		userPresence := &presencev1.UserPresence{
			UserId: userID,
			Online: len(devices) > 0,
		}

		for deviceID := range devices {
			userPresence.DeviceIds = append(userPresence.DeviceIds, deviceID)
		}

		resp.Presences = append(resp.Presences, userPresence)
	}

	return resp, nil
}
