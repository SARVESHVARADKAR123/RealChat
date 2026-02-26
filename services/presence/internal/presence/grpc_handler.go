package presence

import (
	"context"
	"sync"

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
	type result struct {
		presence *presencev1.UserPresence
		index    int
	}
	resChan := make(chan result, len(req.UserIds))
	var wg sync.WaitGroup

	for i, userID := range req.UserIds {
		wg.Add(1)
		go func(idx int, uid string) {
			defer wg.Done()
			devices, err := h.presence.GetUserDevices(ctx, uid)
			if err != nil {
				resChan <- result{&presencev1.UserPresence{UserId: uid, Online: false}, idx}
				return
			}

			userPresence := &presencev1.UserPresence{
				UserId: uid,
				Online: len(devices) > 0,
			}
			for deviceID := range devices {
				userPresence.DeviceIds = append(userPresence.DeviceIds, deviceID)
			}
			resChan <- result{userPresence, idx}
		}(i, userID)
	}

	go func() {
		wg.Wait()
		close(resChan)
	}()

	results := make([]*presencev1.UserPresence, len(req.UserIds))
	for res := range resChan {
		results[res.index] = res.presence
	}
	resp.Presences = results

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
