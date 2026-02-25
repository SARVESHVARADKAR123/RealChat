package grpc

import (
	"context"
	"log"

	profilev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/profile/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/model"
	"github.com/SARVESHVARADKAR123/RealChat/services/profile/internal/service"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	profilev1.UnimplementedProfileApiServer
	svc *service.ProfileService
}

func NewHandler(svc *service.ProfileService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetProfile(ctx context.Context, req *profilev1.GetProfileRequest) (*profilev1.Profile, error) {
	log.Printf("Profile Service GetProfile: userID=%s", req.UserId)
	p, err := h.svc.Get(ctx, req.UserId)
	if err != nil {
		return nil, MapError(err)
	}

	return toProto(p), nil
}

func (h *Handler) UpdateProfile(ctx context.Context, req *profilev1.UpdateProfileRequest) (*profilev1.Profile, error) {
	p, err := h.svc.Get(ctx, req.UserId)
	if err != nil {
		return nil, MapError(err)
	}

	if req.DisplayName != nil {
		p.DisplayName = *req.DisplayName
	}
	if req.AvatarUrl != nil {
		p.AvatarURL = *req.AvatarUrl
	}

	if req.Bio != nil {
		p.Bio = *req.Bio
	}

	if err := h.svc.Update(ctx, p); err != nil {
		return nil, MapError(err)
	}

	return toProto(p), nil
}

func (h *Handler) BatchGetProfiles(ctx context.Context, req *profilev1.BatchGetProfilesRequest) (*profilev1.BatchGetProfilesResponse, error) {
	profiles, err := h.svc.BatchGet(ctx, req.UserIds)
	if err != nil {
		return nil, MapError(err)
	}

	pbProfiles := make([]*profilev1.Profile, 0, len(profiles))
	for _, p := range profiles {
		pbProfiles = append(pbProfiles, toProto(p))
	}

	return &profilev1.BatchGetProfilesResponse{
		Profiles: pbProfiles,
	}, nil
}

func toProto(p *model.Profile) *profilev1.Profile {
	return &profilev1.Profile{
		UserId:      p.UserID,
		DisplayName: p.DisplayName,
		AvatarUrl:   p.AvatarURL,
		Bio:         p.Bio,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
}
