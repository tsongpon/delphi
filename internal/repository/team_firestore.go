package repository

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/tsongpon/delphi/internal/logger"
	"github.com/tsongpon/delphi/internal/model"
	"github.com/tsongpon/delphi/internal/service"
	"go.uber.org/zap"
)

const teamsCollection = "teams"

// Compile-time check that TeamFirestoreRepository implements service.TeamRepository.
var _ service.TeamRepository = (*TeamFirestoreRepository)(nil)

// teamDocument is a Firestore-specific DTO for the Team entity.
type teamDocument struct {
	ID        string    `firestore:"id"`
	Name      string    `firestore:"name"`
	CreatedAt time.Time `firestore:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at"`
}

func toTeamDocument(team *model.Team) *teamDocument {
	return &teamDocument{
		ID:        team.ID,
		Name:      team.Name,
		CreatedAt: team.CreatedAt,
		UpdatedAt: team.UpdatedAt,
	}
}

func toTeamModel(doc *teamDocument) *model.Team {
	return &model.Team{
		ID:        doc.ID,
		Name:      doc.Name,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}
}

// TeamFirestoreRepository is a Firestore-backed implementation of service.TeamRepository.
type TeamFirestoreRepository struct {
	client *firestore.Client
}

// NewTeamFirestoreRepository creates a new TeamFirestoreRepository.
func NewTeamFirestoreRepository(client *firestore.Client) *TeamFirestoreRepository {
	return &TeamFirestoreRepository{client: client}
}

// CreateTeam saves a team to the Firestore "teams" collection.
func (r *TeamFirestoreRepository) CreateTeam(ctx context.Context, team *model.Team) (*model.Team, error) {
	doc := toTeamDocument(team)

	_, err := r.client.Collection(teamsCollection).Doc(doc.ID).Set(ctx, doc)
	if err != nil {
		logger.Error("failed to create team in firestore", zap.Error(err))
		return nil, fmt.Errorf("failed to create team in firestore: %w", err)
	}

	return toTeamModel(doc), nil
}
