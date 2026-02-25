package application

import (
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/repository"
	"github.com/SARVESHVARADKAR123/RealChat/services/conversation/internal/tx"
)

type Service struct {
	repo repository.Repository
	tx   tx.Transactor
}

func New(repo repository.Repository, transactor tx.Transactor) *Service {
	return &Service{repo: repo, tx: transactor}
}
