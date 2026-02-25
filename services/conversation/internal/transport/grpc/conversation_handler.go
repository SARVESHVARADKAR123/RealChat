package grpc

import (
	"context"
	"log/slog"

	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/application"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/auth"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	errActorMismatch  = "actor id mismatch"
	errUserIDMismatch = "user id mismatch"
)

// domainTypeToProto maps the internal domain ConversationType to the proto enum.
func domainTypeToProto(t domain.ConversationType) conversationv1.ConversationType {
	switch t {
	case domain.ConversationDirect:
		return conversationv1.ConversationType_DIRECT
	case domain.ConversationGroup:
		return conversationv1.ConversationType_GROUP
	default:
		return conversationv1.ConversationType_CONVERSATION_TYPE_UNSPECIFIED
	}
}

// protoTypeToDomain maps a proto ConversationType enum to the internal domain type.
func protoTypeToDomain(t conversationv1.ConversationType) (domain.ConversationType, error) {
	switch t {
	case conversationv1.ConversationType_DIRECT:
		return domain.ConversationDirect, nil
	case conversationv1.ConversationType_GROUP:
		return domain.ConversationGroup, nil
	default:
		return "", status.Error(codes.InvalidArgument, "invalid conversation type: must be DIRECT or GROUP")
	}
}

// domainRoleToProto maps internal domain roles to the proto enum.
func domainRoleToProto(r domain.Role) conversationv1.ParticipantRole {
	switch r {
	case domain.RoleAdmin:
		return conversationv1.ParticipantRole_ADMIN
	case domain.RoleMember:
		return conversationv1.ParticipantRole_MEMBER
	default:
		return conversationv1.ParticipantRole_PARTICIPANT_ROLE_UNSPECIFIED
	}
}

func (s *Server) toProtoConversation(conv *domain.Conversation) *conversationv1.Conversation {
	pbParticipants := make([]string, 0, len(conv.Participants))
	pbParticipantsWithRoles := make([]*conversationv1.Participant, 0, len(conv.Participants))

	for uid, p := range conv.Participants {
		pbParticipants = append(pbParticipants, uid)
		pbParticipantsWithRoles = append(pbParticipantsWithRoles, &conversationv1.Participant{
			UserId: uid,
			Role:   domainRoleToProto(p.Role),
		})
	}

	return &conversationv1.Conversation{
		ConversationId:        conv.ID,
		DisplayName:           conv.DisplayName,
		AvatarUrl:             conv.AvatarURL,
		Type:                  domainTypeToProto(conv.Type),
		CreatedAt:             timestamppb.New(conv.CreatedAt),
		ParticipantUserIds:    pbParticipants,
		ParticipantsWithRoles: pbParticipantsWithRoles,
	}
}

func (s *Server) CreateConversation(
	ctx context.Context,
	req *conversationv1.CreateConversationRequest,
) (*conversationv1.CreateConversationResponse, error) {

	userID, err := auth.GetUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if len(req.ParticipantUserIds) > 0 && req.ParticipantUserIds[0] != userID {
		return nil, status.Error(codes.PermissionDenied, errUserIDMismatch)
	}

	convType, err := protoTypeToDomain(req.Type)
	if err != nil {
		return nil, err
	}

	conv, err := s.app.CreateConversation(ctx, application.CreateConversationCommand{
		ID:           req.ConversationId,
		Type:         convType,
		Name:         req.DisplayName,
		AvatarURL:    req.AvatarUrl,
		Participants: req.ParticipantUserIds,
	})
	if err != nil {
		return nil, MapError(err)
	}

	return &conversationv1.CreateConversationResponse{
		Conversation: s.toProtoConversation(conv),
	}, nil
}

func (s *Server) ListConversations(
	ctx context.Context,
	req *conversationv1.ListConversationsRequest,
) (*conversationv1.ListConversationsResponse, error) {

	userID, err := auth.GetUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if req.UserId != userID {
		return nil, status.Error(codes.PermissionDenied, errUserIDMismatch)
	}

	conversations, err := s.app.ListConversations(ctx, userID)
	if err != nil {
		return nil, MapError(err)
	}

	protoConvs := make([]*conversationv1.Conversation, 0, len(conversations))
	for _, conv := range conversations {
		protoConvs = append(protoConvs, s.toProtoConversation(conv))
	}

	return &conversationv1.ListConversationsResponse{
		Conversations: protoConvs,
	}, nil
}

func (s *Server) GetConversation(
	ctx context.Context,
	req *conversationv1.GetConversationRequest,
) (*conversationv1.GetConversationResponse, error) {

	conv, err := s.app.GetConversation(ctx, req.ConversationId)
	if err != nil {
		return nil, MapError(err)
	}

	pbConv := s.toProtoConversation(conv)
	slog.Info("GetConversation response", "conv_id", conv.ID, "participants", pbConv.ParticipantUserIds)
	return &conversationv1.GetConversationResponse{
		Conversation:       pbConv,
		ParticipantUserIds: pbConv.ParticipantUserIds,
	}, nil
}

func (s *Server) AddParticipant(
	ctx context.Context,
	req *conversationv1.AddParticipantRequest,
) (*conversationv1.AddParticipantResponse, error) {

	userID, err := auth.GetUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if req.ActorUserId != userID {
		return nil, status.Error(codes.PermissionDenied, errActorMismatch)
	}

	err = s.app.AddParticipant(ctx, application.AddParticipantCommand{
		ConversationID: req.ConversationId,
		ActorID:        req.ActorUserId,
		TargetID:       req.TargetUserId,
	})
	if err != nil {
		return nil, MapError(err)
	}

	return &conversationv1.AddParticipantResponse{}, nil
}

func (s *Server) RemoveParticipant(
	ctx context.Context,
	req *conversationv1.RemoveParticipantRequest,
) (*conversationv1.RemoveParticipantResponse, error) {

	userID, err := auth.GetUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if req.ActorUserId != userID {
		return nil, status.Error(codes.PermissionDenied, errActorMismatch)
	}

	err = s.app.RemoveParticipant(ctx, application.RemoveParticipantCommand{
		ConversationID: req.ConversationId,
		ActorID:        req.ActorUserId,
		TargetID:       req.TargetUserId,
	})
	if err != nil {
		return nil, MapError(err)
	}

	return &conversationv1.RemoveParticipantResponse{}, nil
}

func (s *Server) UpdateReadReceipt(
	ctx context.Context,
	req *conversationv1.UpdateReadReceiptRequest,
) (*conversationv1.UpdateReadReceiptResponse, error) {

	userID, err := auth.GetUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if req.UserId != userID {
		return nil, status.Error(codes.PermissionDenied, errUserIDMismatch)
	}

	err = s.app.UpdateReadReceipt(
		ctx,
		req.ConversationId,
		req.UserId,
		req.ReadSequence,
	)
	if err != nil {
		return nil, MapError(err)
	}

	return &conversationv1.UpdateReadReceiptResponse{}, nil
}

// NextSequence is an internal RPC called by the message service to atomically
// claim the next message sequence number for a conversation.
// No user-auth check — this is a trusted internal peer call.
func (s *Server) NextSequence(
	ctx context.Context,
	req *conversationv1.NextSequenceRequest,
) (*conversationv1.NextSequenceResponse, error) {

	if req.ConversationId == "" {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is required")
	}

	seq, err := s.app.NextSequence(ctx, req.ConversationId)
	if err != nil {
		return nil, MapError(err)
	}

	return &conversationv1.NextSequenceResponse{Sequence: seq}, nil
}
