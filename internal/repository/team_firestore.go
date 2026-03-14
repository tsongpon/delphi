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
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const teamsCollection = "teams"

// Compile-time check that TeamFirestoreRepository implements service.TeamRepository.
var _ service.TeamRepository = (*TeamFirestoreRepository)(nil)

// teamDocument is a Firestore-specific DTO for the Team entity.
type teamDocument struct {
	ID        string    `firestore:"id"`
	Name      string    `firestore:"name"`
	CreatedBy string    `firestore:"created_by"`
	CreatedAt time.Time `firestore:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at"`
}

func toTeamDocument(team *model.Team) *teamDocument {
	return &teamDocument{
		ID:        team.ID,
		Name:      team.Name,
		CreatedBy: team.CreatedBy,
		CreatedAt: team.CreatedAt,
		UpdatedAt: team.UpdatedAt,
	}
}

func toTeamModel(doc *teamDocument) *model.Team {
	return &model.Team{
		ID:        doc.ID,
		Name:      doc.Name,
		CreatedBy: doc.CreatedBy,
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

// GetTeamByID fetches a team by its document ID.
func (r *TeamFirestoreRepository) GetTeamByID(ctx context.Context, teamID string) (*model.Team, error) {
	docSnap, err := r.client.Collection(teamsCollection).Doc(teamID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		logger.Error("failed to get team by id", zap.String("team_id", teamID), zap.Error(err))
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	var doc teamDocument
	if err := docSnap.DataTo(&doc); err != nil {
		return nil, fmt.Errorf("failed to deserialize team document: %w", err)
	}
	return toTeamModel(&doc), nil
}

// GetTeamByName returns the first team whose name matches exactly, or nil if none.
func (r *TeamFirestoreRepository) GetTeamByName(ctx context.Context, name string) (*model.Team, error) {
	iter := r.client.Collection(teamsCollection).
		Where("name", "==", name).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	docSnap, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		logger.Error("failed to get team by name", zap.String("name", name), zap.Error(err))
		return nil, fmt.Errorf("failed to get team by name: %w", err)
	}

	var doc teamDocument
	if err := docSnap.DataTo(&doc); err != nil {
		return nil, fmt.Errorf("failed to deserialize team document: %w", err)
	}
	return toTeamModel(&doc), nil
}
