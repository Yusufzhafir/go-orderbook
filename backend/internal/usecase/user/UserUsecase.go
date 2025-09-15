package user

import (
	"context"
	"errors"

	repository "github.com/Yusufzhafir/go-orderbook/backend/internal/repository/user"
)

type UserUseCase interface {
	Register(ctx context.Context, username, password string) (int64, error)
	Login(ctx context.Context, username, password string) (*repository.User, error)
	GetProfile(ctx context.Context, userID int64) (*repository.User, error)
}

type userUseCaseImpl struct {
	repo repository.UserRepository
}

func NewUserUseCase(repo repository.UserRepository) UserUseCase {
	return &userUseCaseImpl{repo: repo}
}

func (uc *userUseCaseImpl) Register(ctx context.Context, username, password string) (int64, error) {
	// check duplicate username
	existing, _ := uc.repo.GetByUsername(ctx, username)
	if existing != nil {
		return 0, errors.New("username already exists")
	}
	return uc.repo.Create(ctx, username, password)
}

func (uc *userUseCaseImpl) Login(ctx context.Context, username, password string) (*repository.User, error) {
	return uc.repo.VerifyPassword(ctx, username, password)
}

func (uc *userUseCaseImpl) GetProfile(ctx context.Context, userID int64) (*repository.User, error) {
	return uc.repo.GetByID(ctx, userID)
}
