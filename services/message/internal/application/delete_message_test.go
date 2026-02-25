package application

import (
	"context"
	"database/sql"
	"testing"
	"time"

	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockRepo is a mock for the Repository interface
type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) InsertMessage(ctx context.Context, tx *sql.Tx, msg *domain.Message) error {
	return m.Called(ctx, tx, msg).Error(0)
}
func (m *MockRepo) MarkMessageDeleted(ctx context.Context, tx *sql.Tx, msgID string) error {
	return m.Called(ctx, tx, msgID).Error(0)
}
func (m *MockRepo) GetMessageForUpdate(ctx context.Context, tx *sql.Tx, messageID string) (*domain.Message, error) {
	args := m.Called(ctx, tx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Message), args.Error(1)
}
func (m *MockRepo) FetchMessages(ctx context.Context, convID string, lastSeq int64, limit int) ([]*domain.Message, error) {
	args := m.Called(ctx, convID, lastSeq, limit)
	return args.Get(0).([]*domain.Message), args.Error(1)
}
func (m *MockRepo) TryInsertIdempotency(ctx context.Context, tx *sql.Tx, key, userID, conversationID string, expiresAt time.Time) (bool, error) {
	return true, nil
}
func (m *MockRepo) GetIdempotencyForUpdate(ctx context.Context, tx *sql.Tx, key, userID, conversationID string) ([]byte, error) {
	return nil, nil
}
func (m *MockRepo) UpdateIdempotencyResponse(ctx context.Context, tx *sql.Tx, key, userID, conversationID string, payload []byte) error {
	return nil
}
func (m *MockRepo) InsertOutbox(ctx context.Context, tx *sql.Tx, aggregateType, aggregateID, eventType string, payload []byte) error {
	return m.Called(ctx, tx, aggregateType, aggregateID, eventType, payload).Error(0)
}

// MockConvClient is a mock for the ConversationApiClient interface
type MockConvClient struct {
	mock.Mock
	conversationv1.ConversationApiClient
}

func (m *MockConvClient) GetConversation(ctx context.Context, req *conversationv1.GetConversationRequest, opts ...grpc.CallOption) (*conversationv1.GetConversationResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*conversationv1.GetConversationResponse), args.Error(1)
}

// MockTransactor is a mock for the Transactor interface
type MockTransactor struct{}

func (m *MockTransactor) WithTx(ctx context.Context, fn func(ctx context.Context, tx *sql.Tx) error) error {
	return fn(ctx, nil)
}

func TestDeleteMessage_Admin(t *testing.T) {
	ctx := context.Background()
	repo := new(MockRepo)
	tx := new(MockTransactor)
	convSvc := new(MockConvClient)
	svc := &Service{repo: repo, tx: tx, convSvc: convSvc}

	convID := "conv-1"
	msgID := "msg-1"
	senderID := "user-sender"
	adminID := "user-admin"
	otherID := "user-other"

	msg := &domain.Message{
		ID:             msgID,
		ConversationID: convID,
		SenderID:       senderID,
	}

	t.Run("Sender can delete", func(t *testing.T) {
		repo.On("GetMessageForUpdate", ctx, mock.Anything, msgID).Return(msg, nil).Once()
		repo.On("MarkMessageDeleted", ctx, mock.Anything, msgID).Return(nil).Once()
		repo.On("InsertOutbox", ctx, mock.Anything, "message", convID, "MESSAGE_DELETED", mock.Anything).Return(nil).Once()

		err := svc.DeleteMessage(ctx, DeleteMessageCommand{
			ConversationID: convID,
			MessageID:      msgID,
			RequesterID:    senderID,
		})
		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("Admin can delete others message", func(t *testing.T) {
		repo.On("GetMessageForUpdate", ctx, mock.Anything, msgID).Return(msg, nil).Once()
		convSvc.On("GetConversation", ctx, &conversationv1.GetConversationRequest{ConversationId: convID}).Return(&conversationv1.GetConversationResponse{
			Conversation: &conversationv1.Conversation{
				ParticipantsWithRoles: []*conversationv1.Participant{
					{UserId: adminID, Role: conversationv1.ParticipantRole_ADMIN},
				},
			},
		}, nil).Once()
		repo.On("MarkMessageDeleted", ctx, mock.Anything, msgID).Return(nil).Once()
		repo.On("InsertOutbox", ctx, mock.Anything, "message", convID, "MESSAGE_DELETED", mock.Anything).Return(nil).Once()

		err := svc.DeleteMessage(ctx, DeleteMessageCommand{
			ConversationID: convID,
			MessageID:      msgID,
			RequesterID:    adminID,
		})
		assert.NoError(t, err)
		repo.AssertExpectations(t)
		convSvc.AssertExpectations(t)
	})

	t.Run("Other member cannot delete", func(t *testing.T) {
		repo.On("GetMessageForUpdate", ctx, mock.Anything, msgID).Return(msg, nil).Once()
		convSvc.On("GetConversation", ctx, &conversationv1.GetConversationRequest{ConversationId: convID}).Return(&conversationv1.GetConversationResponse{
			Conversation: &conversationv1.Conversation{
				ParticipantsWithRoles: []*conversationv1.Participant{
					{UserId: otherID, Role: conversationv1.ParticipantRole_MEMBER},
				},
			},
		}, nil).Once()

		err := svc.DeleteMessage(ctx, DeleteMessageCommand{
			ConversationID: convID,
			MessageID:      msgID,
			RequesterID:    otherID,
		})
		assert.ErrorIs(t, err, domain.ErrNotParticipant)
		repo.AssertExpectations(t)
		convSvc.AssertExpectations(t)
	})
}
