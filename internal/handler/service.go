package handler

import (
	"context"

	"github.com/tsongpon/delphi/internal/model"
)

type UserService interface {
	RegisterUser(ctx context.Context, user *model.User) (*model.User, error)
	LoginUser(ctx context.Context, email, password string) (string, error)
}
