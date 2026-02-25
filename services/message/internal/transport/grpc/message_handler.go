package grpc

import (
	"context"

	messagev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/message/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/application"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) SendMessage(
	ctx context.Context,
	req *messagev1.SendMessageRequest,
) (*messagev1.SendMessageResponse, error) {

	userID, err := auth.GetUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if req.SenderUserId != userID {
		return nil, status.Error(codes.PermissionDenied, "sender id mismatch")
	}

	msg, err := s.app.SendMessage(ctx, application.SendMessageCommand{
		ConversationID: req.ConversationId,
		UserID:         req.SenderUserId,
		ClientMsgID:    req.IdempotencyKey,
		Type:           req.MessageType,
		Content:        req.Content,
		Metadata:       req.MetadataJson,
	})
	if err != nil {
		return nil, MapError(err)
	}

	return &messagev1.SendMessageResponse{
		Message: &messagev1.Message{
			MessageId:      msg.ID,
			ConversationId: msg.ConversationID,
			SenderUserId:   msg.SenderID,
			Sequence:       msg.Sequence,
			MessageType:    msg.Type,
			Content:        msg.Content,
			MetadataJson:   msg.Metadata,
			SentAt:         timestamppb.New(msg.SentAt),
		},
	}, nil
}

func (s *Server) DeleteMessage(
	ctx context.Context,
	req *messagev1.DeleteMessageRequest,
) (*messagev1.DeleteMessageResponse, error) {

	userID, err := auth.GetUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if req.ActorUserId != userID {
		return nil, status.Error(codes.PermissionDenied, "actor id mismatch")
	}

	err = s.app.DeleteMessage(ctx, application.DeleteMessageCommand{
		ConversationID: req.ConversationId,
		MessageID:      req.MessageId,
		RequesterID:    req.ActorUserId,
	})
	if err != nil {
		return nil, MapError(err)
	}

	return &messagev1.DeleteMessageResponse{}, nil
}

func (s *Server) SyncMessages(
	ctx context.Context,
	req *messagev1.SyncMessagesRequest,
) (*messagev1.SyncMessagesResponse, error) {

	userID, err := auth.GetUserID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	messages, err := s.app.SyncMessages(
		ctx,
		req.ConversationId,
		userID,
		req.AfterSequence,
		int(req.PageSize),
	)
	if err != nil {
		return nil, MapError(err)
	}

	var protoMsgs []*messagev1.Message

	for _, m := range messages {
		pm := &messagev1.Message{
			MessageId:      m.ID,
			ConversationId: m.ConversationID,
			SenderUserId:   m.SenderID,
			Sequence:       m.Sequence,
			MessageType:    m.Type,
			Content:        m.Content,
			MetadataJson:   m.Metadata,
			SentAt:         timestamppb.New(m.SentAt),
		}
		if m.DeletedAt != nil {
			pm.DeletedAt = timestamppb.New(*m.DeletedAt)
		}
		protoMsgs = append(protoMsgs, pm)
	}

	return &messagev1.SyncMessagesResponse{
		Messages: protoMsgs,
	}, nil
}
