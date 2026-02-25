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

func (h *GRPCHandler) RegisterSession(ctx context.Context, req *presencev1.RegisterSessionRequest) (*presencev1.RegisterSessionResponse, error) {
	if err := h.presence.Register(ctx, req.UserId, req.DeviceId, req.InstanceId); err != nil {
		return nil, err
	}
	return &presencev1.RegisterSessionResponse{}, nil
}

func (h *GRPCHandler) UnregisterSession(ctx context.Context, req *presencev1.UnregisterSessionRequest) (*presencev1.UnregisterSessionResponse, error) {
	if err := h.presence.Unregister(ctx, req.UserId, req.DeviceId); err != nil {
		return nil, err
	}
	return &presencev1.UnregisterSessionResponse{}, nil
}

func (h *GRPCHandler) RefreshSession(ctx context.Context, req *presencev1.RefreshSessionRequest) (*presencev1.RefreshSessionResponse, error) {
	if err := h.presence.Refresh(ctx, req.UserId, req.DeviceId); err != nil {
		return nil, err
	}
	return &presencev1.RefreshSessionResponse{}, nil
}

func (h *GRPCHandler) GetUserDevices(ctx context.Context, req *presencev1.GetUserDevicesRequest) (*presencev1.GetUserDevicesResponse, error) {
	devices, err := h.presence.GetUserDevices(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	resp := &presencev1.GetUserDevicesResponse{}
	for deviceID, instanceID := range devices {
		resp.Devices = append(resp.Devices, &presencev1.DeviceInfo{
			DeviceId:   deviceID,
			InstanceId: instanceID,
		})
	}

	return resp, nil
}
