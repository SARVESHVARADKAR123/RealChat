package clients

import (
	authv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/auth/v1"
	messagingv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/messaging/v1"
	presencev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/presence/v1"
	profilev1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/profile/v1"
	"google.golang.org/grpc"
)

type Factory struct {
	Auth      authv1.AuthApiClient
	Profile   profilev1.ProfileApiClient
	Messaging messagingv1.MessagingApiClient
	Presence  presencev1.PresenceApiClient
}

func NewFactory(a, p, m, d *grpc.ClientConn) *Factory {
	return &Factory{
		Auth:      authv1.NewAuthApiClient(a),
		Profile:   profilev1.NewProfileApiClient(p),
		Messaging: messagingv1.NewMessagingApiClient(m),
		Presence:  presencev1.NewPresenceApiClient(d),
	}
}
