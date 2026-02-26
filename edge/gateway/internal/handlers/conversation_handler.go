package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"strings"

	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	profilev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/profile/v1"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/middleware"
	"github.com/SARVESHVARADKAR123/RealChat/edge/gateway/internal/transport"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	errInvalidBody   = "invalid_body"
	errMissingParams = "missing_params"
	errInternalError = "internal_error"
	errInvalidType   = "invalid_type"
	errMissingConvID = "missing_conv_id"
	errInvalidSeq    = "invalid_sequence"
	msgInvalidJSON   = "invalid json"
)

// ConversationHandler handles all routes that talk to the conversation service.
type ConversationHandler struct {
	client  conversationv1.ConversationApiClient
	profile profilev1.ProfileApiClient
}

func NewConversationHandler(c conversationv1.ConversationApiClient, p profilev1.ProfileApiClient) *ConversationHandler {
	return &ConversationHandler{client: c, profile: p}
}

// CreateConversation POST /api/conversations
func (h *ConversationHandler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	reqID := middleware.RequestIDFromContext(r.Context())

	var req struct {
		ID           string   `json:"conversation_id"`
		Type         string   `json:"type"`
		DisplayName  string   `json:"display_name"`
		AvatarURL    string   `json:"avatar_url"`
		Participants []string `json:"participant_user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, errInvalidBody, msgInvalidJSON)
		return
	}

	// Normalize and flatten participants (handle CSV-like strings and trim whitespace)
	var participants []string
	seen := make(map[string]struct{})

	// Add the current user first
	participants = append(participants, userID)
	seen[userID] = struct{}{}

	for _, p := range req.Participants {
		// Split by comma in case Postman/User sent ["id1,id2"]
		parts := strings.Split(p, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			if _, ok := seen[trimmed]; !ok {
				participants = append(participants, trimmed)
				seen[trimmed] = struct{}{}
			}
		}
	}

	convID := req.ID
	if convID == "" {
		convID = uuid.NewString()
	}

	var pbType conversationv1.ConversationType
	normalizedType := strings.ToLower(req.Type)
	if normalizedType == "group" {
		pbType = conversationv1.ConversationType_GROUP
	} else {
		pbType = conversationv1.ConversationType_DIRECT
	}

	slog.Info("Creating conversation", "conv_id", convID, "req_type", req.Type, "mapped_type", pbType.String(), "participants", participants)

	ctx, cancel := transport.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	ctx = transport.WithMeta(ctx, userID, reqID)

	resp, err := h.client.CreateConversation(ctx, &conversationv1.CreateConversationRequest{
		ConversationId:     convID,
		Type:               pbType,
		DisplayName:        req.DisplayName,
		AvatarUrl:          req.AvatarURL,
		ParticipantUserIds: participants,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	h.resolveConversation(ctx, userID, resp.Conversation)
	transport.WriteJSON(w, http.StatusCreated, resp)
}

// ListConversations GET /api/conversations
func (h *ConversationHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	reqID := middleware.RequestIDFromContext(r.Context())

	ctx, cancel := transport.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	ctx = transport.WithMeta(ctx, userID, reqID)

	resp, err := h.client.ListConversations(ctx, &conversationv1.ListConversationsRequest{
		UserId: userID,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	h.resolveConversations(ctx, userID, resp.Conversations)
	transport.WriteJSON(w, http.StatusOK, resp)
}

// GetConversation GET /api/conversations/{id}
func (h *ConversationHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	reqID := middleware.RequestIDFromContext(r.Context())

	// Try path parameter first (from chi router), then fallback to query param
	convID := chi.URLParam(r, "id")
	if convID == "" {
		convID = r.URL.Query().Get("id")
	}

	if convID == "" {
		transport.WriteError(w, http.StatusBadRequest, errMissingConvID, "id is required (as path param or query param)")
		return
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	ctx = transport.WithMeta(ctx, userID, reqID)

	resp, err := h.client.GetConversation(ctx, &conversationv1.GetConversationRequest{
		ConversationId: convID,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	h.resolveConversation(ctx, userID, resp.Conversation)
	transport.WriteJSON(w, http.StatusOK, resp)
}

// AddParticipant POST /api/participants
func (h *ConversationHandler) AddParticipant(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	reqID := middleware.RequestIDFromContext(r.Context())

	var req struct {
		ConversationID string `json:"conversation_id"`
		TargetUserID   string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, errInvalidBody, msgInvalidJSON)
		return
	}
	if req.ConversationID == "" || req.TargetUserID == "" {
		transport.WriteError(w, http.StatusBadRequest, errMissingParams, "conversation_id and user_id are required")
		return
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	ctx = transport.WithMeta(ctx, userID, reqID)

	_, err := h.client.AddParticipant(ctx, &conversationv1.AddParticipantRequest{
		ConversationId: req.ConversationID,
		ActorUserId:    userID,
		TargetUserId:   req.TargetUserID,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// RemoveParticipant DELETE /api/participants
func (h *ConversationHandler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	reqID := middleware.RequestIDFromContext(r.Context())

	var req struct {
		ConversationID string `json:"conversation_id"`
		TargetUserID   string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, errInvalidBody, msgInvalidJSON)
		return
	}
	if req.ConversationID == "" || req.TargetUserID == "" {
		transport.WriteError(w, http.StatusBadRequest, "missing_params", "conversation_id and user_id are required")
		return
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	ctx = transport.WithMeta(ctx, userID, reqID)

	_, err := h.client.RemoveParticipant(ctx, &conversationv1.RemoveParticipantRequest{
		ConversationId: req.ConversationID,
		ActorUserId:    userID,
		TargetUserId:   req.TargetUserID,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ReadReceipt POST /api/read-receipt
func (h *ConversationHandler) ReadReceipt(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	reqID := middleware.RequestIDFromContext(r.Context())

	var req struct {
		ConversationID string `json:"conversation_id"`
		Sequence       int64  `json:"sequence"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transport.WriteError(w, http.StatusBadRequest, errInvalidBody, msgInvalidJSON)
		return
	}
	if req.ConversationID == "" {
		transport.WriteError(w, http.StatusBadRequest, errMissingConvID, "conversation_id is required")
		return
	}
	if req.Sequence < 0 {
		transport.WriteError(w, http.StatusBadRequest, errInvalidSeq, "sequence must be >= 0")
		return
	}

	ctx, cancel := transport.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	ctx = transport.WithMeta(ctx, userID, reqID)

	_, err := h.client.UpdateReadReceipt(ctx, &conversationv1.UpdateReadReceiptRequest{
		UserId:         userID,
		ConversationId: req.ConversationID,
		ReadSequence:   req.Sequence,
	})
	if err != nil {
		transport.GRPCError(w, err)
		return
	}

	slog.Debug("read receipt updated", "user_id", userID, "conversation_id", req.ConversationID, "sequence", req.Sequence)
	transport.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ConversationHandler) resolveConversations(ctx context.Context, currentUserID string, convs []*conversationv1.Conversation) {
	// 1. Identify direct conversations and the "other" user IDs
	otherUserIDs := h.collectOtherUserIDs(currentUserID, convs)
	if len(otherUserIDs) == 0 {
		return
	}

	// 2. Fetch profiles in batch
	profileMap := h.fetchProfiles(ctx, otherUserIDs)
	if len(profileMap) == 0 {
		return
	}

	// 3. Populate display names and avatars
	h.applyProfiles(currentUserID, convs, profileMap)
}

func (h *ConversationHandler) collectOtherUserIDs(currentUserID string, convs []*conversationv1.Conversation) []string {
	otherUserIDs := make(map[string]struct{})
	for _, c := range convs {
		if c.Type == conversationv1.ConversationType_DIRECT {
			for _, pid := range c.ParticipantUserIds {
				if pid != currentUserID {
					otherUserIDs[pid] = struct{}{}
				}
			}
		}
	}

	uids := make([]string, 0, len(otherUserIDs))
	for id := range otherUserIDs {
		uids = append(uids, id)
	}
	return uids
}

func (h *ConversationHandler) fetchProfiles(ctx context.Context, uids []string) map[string]*profilev1.Profile {
	resp, err := h.profile.BatchGetProfiles(ctx, &profilev1.BatchGetProfilesRequest{UserIds: uids})
	if err != nil {
		slog.Error("failed to batch get profiles", "error", err)
		return nil
	}

	profileMap := make(map[string]*profilev1.Profile)
	for _, p := range resp.Profiles {
		profileMap[p.UserId] = p
	}
	return profileMap
}

func (h *ConversationHandler) applyProfiles(currentUserID string, convs []*conversationv1.Conversation, profileMap map[string]*profilev1.Profile) {
	for _, c := range convs {
		if c.Type != conversationv1.ConversationType_DIRECT {
			continue
		}
		for _, pid := range c.ParticipantUserIds {
			if pid != currentUserID {
				if p, ok := profileMap[pid]; ok {
					c.DisplayName = p.DisplayName
					c.AvatarUrl = p.AvatarUrl
				}
				break
			}
		}
	}
}

func (h *ConversationHandler) resolveConversation(ctx context.Context, currentUserID string, c *conversationv1.Conversation) {
	h.resolveConversations(ctx, currentUserID, []*conversationv1.Conversation{c})
}
