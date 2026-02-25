package application

import (
	conversationv1 "github.com/SARVESHVARADKAR123/RealChat/contracts/gen/go/conversation/v1"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/repository"
	"github.com/SARVESHVARADKAR123/RealChat/services/message/internal/tx"
	"go.uber.org/zap"
)

type Service struct {
	repo    repository.Repository
	tx      tx.Transactor
	convSvc conversationv1.ConversationApiClient
	log     *zap.Logger
}

func New(repo repository.Repository, transactor tx.Transactor, convSvc conversationv1.ConversationApiClient, log *zap.Logger) *Service {
	return &Service{repo: repo, tx: transactor, convSvc: convSvc, log: log}
}
