package clients

import (
	authv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/auth/v1"
	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	messagev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/message/v1"
	presencev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/presence/v1"
	profilev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/profile/v1"
	"google.golang.org/grpc"
)

type Factory struct {
	Auth         authv1.AuthApiClient
	Profile      profilev1.ProfileApiClient
	Conversation conversationv1.ConversationApiClient
	Message      messagev1.MessageApiClient
	Presence     presencev1.PresenceApiClient
}

func NewFactory(a, p, c, m, pr *grpc.ClientConn) *Factory {
	return &Factory{
		Auth:         authv1.NewAuthApiClient(a),
		Profile:      profilev1.NewProfileApiClient(p),
		Conversation: conversationv1.NewConversationApiClient(c),
		Message:      messagev1.NewMessageApiClient(m),
		Presence:     presencev1.NewPresenceApiClient(pr),
	}
}
