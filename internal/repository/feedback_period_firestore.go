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

const feedbackPeriodsCollection = "feedback_periods"

// Compile-time check that FeedbackPeriodFirestoreRepository implements service.FeedbackPeriodRepository.
var _ service.FeedbackPeriodRepository = (*FeedbackPeriodFirestoreRepository)(nil)

type feedbackPeriodDocument struct {
	ID        string `firestore:"id"`
	TeamID    string `firestore:"team_id"`
	Name      string `firestore:"name"`
	StartDate int64  `firestore:"start_date"` // Unix timestamp
	EndDate   int64  `firestore:"end_date"`   // Unix timestamp
	CreatedBy string `firestore:"created_by"`
	CreatedAt int64  `firestore:"created_at"` // Unix timestamp
	UpdatedAt int64  `firestore:"updated_at"` // Unix timestamp
}

func toFeedbackPeriodDocument(p *model.FeedbackPeriod) *feedbackPeriodDocument {
	return &feedbackPeriodDocument{
		ID:        p.ID,
		TeamID:    p.TeamID,
		Name:      p.Name,
		StartDate: p.StartDate.Unix(),
		EndDate:   p.EndDate.Unix(),
		CreatedBy: p.CreatedBy,
		CreatedAt: p.CreatedAt.Unix(),
		UpdatedAt: p.UpdatedAt.Unix(),
	}
}

func toFeedbackPeriodModel(doc *feedbackPeriodDocument) *model.FeedbackPeriod {
	return &model.FeedbackPeriod{
		ID:        doc.ID,
		TeamID:    doc.TeamID,
		Name:      doc.Name,
		StartDate: time.Unix(doc.StartDate, 0).UTC(),
		EndDate:   time.Unix(doc.EndDate, 0).UTC(),
		CreatedBy: doc.CreatedBy,
		CreatedAt: time.Unix(doc.CreatedAt, 0).UTC(),
		UpdatedAt: time.Unix(doc.UpdatedAt, 0).UTC(),
	}
}

type FeedbackPeriodFirestoreRepository struct {
	client *firestore.Client
}

func NewFeedbackPeriodFirestoreRepository(client *firestore.Client) *FeedbackPeriodFirestoreRepository {
	return &FeedbackPeriodFirestoreRepository{client: client}
}

func (r *FeedbackPeriodFirestoreRepository) CreatePeriod(ctx context.Context, period *model.FeedbackPeriod) (*model.FeedbackPeriod, error) {
	doc := toFeedbackPeriodDocument(period)
	_, err := r.client.Collection(feedbackPeriodsCollection).Doc(doc.ID).Set(ctx, doc)
	if err != nil {
		logger.Error("failed to create feedback period", zap.Error(err))
		return nil, fmt.Errorf("failed to create feedback period: %w", err)
	}
	return period, nil
}

// GetActivePeriodForTeam returns the active period for a team at the given time.
// It queries for periods where start_date <= now and team_id == teamID, then
// filters end_date >= now in memory to avoid a Firestore composite index on two range fields.
func (r *FeedbackPeriodFirestoreRepository) GetActivePeriodForTeam(ctx context.Context, teamID string, now time.Time) (*model.FeedbackPeriod, error) {
	nowUnix := now.Unix()
	iter := r.client.Collection(feedbackPeriodsCollection).
		Where("team_id", "==", teamID).
		Where("start_date", "<=", nowUnix).
		OrderBy("start_date", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query active period: %w", err)
		}
		var doc feedbackPeriodDocument
		if err := docSnap.DataTo(&doc); err != nil {
			continue
		}
		if doc.EndDate >= nowUnix {
			return toFeedbackPeriodModel(&doc), nil
		}
	}
	return nil, nil
}

func (r *FeedbackPeriodFirestoreRepository) ListPeriodsForTeam(ctx context.Context, teamID string) ([]*model.FeedbackPeriod, error) {
	iter := r.client.Collection(feedbackPeriodsCollection).
		Where("team_id", "==", teamID).
		OrderBy("start_date", firestore.Desc).
		Documents(ctx)
	defer iter.Stop()

	var periods []*model.FeedbackPeriod
	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list periods: %w", err)
		}
		var doc feedbackPeriodDocument
		if err := docSnap.DataTo(&doc); err != nil {
			continue
		}
		periods = append(periods, toFeedbackPeriodModel(&doc))
	}
	if periods == nil {
		return []*model.FeedbackPeriod{}, nil
	}
	return periods, nil
}

func (r *FeedbackPeriodFirestoreRepository) DeletePeriod(ctx context.Context, teamID, periodID string) error {
	docRef := r.client.Collection(feedbackPeriodsCollection).Doc(periodID)
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("period not found")
		}
		return fmt.Errorf("failed to get period: %w", err)
	}
	var doc feedbackPeriodDocument
	if err := docSnap.DataTo(&doc); err != nil {
		return fmt.Errorf("failed to read period: %w", err)
	}
	if doc.TeamID != teamID {
		return fmt.Errorf("period not found")
	}
	if _, err := docRef.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete period: %w", err)
	}
	return nil
}
