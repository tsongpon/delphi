package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tsongpon/delphi/internal/logger"
	"github.com/tsongpon/delphi/internal/model"
	"go.uber.org/zap"
)

type TeamService interface {
	CreateTeam(ctx context.Context, name string) (*model.Team, error)
}

type TeamServiceImpl struct {
	repo TeamRepository
}

func NewTeamService(repo TeamRepository) *TeamServiceImpl {
	return &TeamServiceImpl{repo: repo}
}

func (s *TeamServiceImpl) CreateTeam(ctx context.Context, name string) (*model.Team, error) {
	now := time.Now()
	team := &model.Team{
		ID:        uuid.New().String(),
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := s.repo.CreateTeam(ctx, team)
	if err != nil {
		logger.Error("failed to create team", zap.Error(err))
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	return created, nil
}
