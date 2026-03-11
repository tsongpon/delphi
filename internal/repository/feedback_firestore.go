package repository

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/tsongpon/delphi/internal/logger"
	"github.com/tsongpon/delphi/internal/model"
	"github.com/tsongpon/delphi/internal/service"
	"go.uber.org/zap"
)

const feedbacksCollection = "feedbacks"

// Compile-time check that FeedbackFirestoreRepository implements service.FeedbackRepository.
var _ service.FeedbackRepository = (*FeedbackFirestoreRepository)(nil)

type feedbackDocument struct {
	ID                 string    `firestore:"id"`
	Period             string    `firestore:"period"`
	RevieweeID         string    `firestore:"reviewee_id"`
	ReviewerID         string    `firestore:"reviewer_id"`
	CommunicationScore int       `firestore:"communication_score"`
	LeadershipScore    int       `firestore:"leadership_score"`
	TechnicalScore     int       `firestore:"technical_score"`
	CollaborationScore int       `firestore:"collaboration_score"`
	DeliveryScore      int       `firestore:"delivery_score"`
	StrengthsComment   string    `firestore:"strengths_comment"`
	WeaknessesComment  string    `firestore:"weaknesses_comment"`
	Visibility         string    `firestore:"visibility"`
	CreatedAt          time.Time `firestore:"created_at"`
	UpdatedAt          time.Time `firestore:"updated_at"`
}

func toFeedbackDocument(f *model.Feedback) *feedbackDocument {
	return &feedbackDocument{
		ID:                 f.ID,
		Period:             f.Period,
		RevieweeID:         f.RevieweeID,
		ReviewerID:         f.ReviewerID,
		CommunicationScore: f.CommunicationScore,
		LeadershipScore:    f.LeadershipScore,
		TechnicalScore:     f.TechnicalScore,
		CollaborationScore: f.CollaborationScore,
		DeliveryScore:      f.DeliveryScore,
		StrengthsComment:   f.StrengthsComment,
		WeaknessesComment:  f.WeaknessesComment,
		Visibility:         f.Visibility,
		CreatedAt:          f.CreatedAt,
		UpdatedAt:          f.UpdatedAt,
	}
}

func toFeedbackModel(doc *feedbackDocument) *model.Feedback {
	return &model.Feedback{
		ID:                 doc.ID,
		Period:             doc.Period,
		RevieweeID:         doc.RevieweeID,
		ReviewerID:         doc.ReviewerID,
		CommunicationScore: doc.CommunicationScore,
		LeadershipScore:    doc.LeadershipScore,
		TechnicalScore:     doc.TechnicalScore,
		CollaborationScore: doc.CollaborationScore,
		DeliveryScore:      doc.DeliveryScore,
		StrengthsComment:   doc.StrengthsComment,
		WeaknessesComment:  doc.WeaknessesComment,
		Visibility:         doc.Visibility,
		CreatedAt:          doc.CreatedAt,
		UpdatedAt:          doc.UpdatedAt,
	}
}

type FeedbackFirestoreRepository struct {
	client *firestore.Client
}

func NewFeedbackFirestoreRepository(client *firestore.Client) *FeedbackFirestoreRepository {
	return &FeedbackFirestoreRepository{client: client}
}

func (r *FeedbackFirestoreRepository) CreateFeedback(ctx context.Context, feedback *model.Feedback) (*model.Feedback, error) {
	doc := toFeedbackDocument(feedback)

	docRef := r.client.Collection(feedbacksCollection).Doc(doc.ID)
	_, err := docRef.Set(ctx, doc)
	if err != nil {
		logger.Error("failed to create feedback in firestore", zap.Error(err))
		return nil, fmt.Errorf("failed to create feedback in firestore: %w", err)
	}

	return feedback, nil
}

func (r *FeedbackFirestoreRepository) GetFeedbacksByRevieweeID(ctx context.Context, revieweeID string, limit int, cursor string) ([]*model.Feedback, error) {
	q := r.client.Collection(feedbacksCollection).
		Where("reviewee_id", "==", revieweeID).
		OrderBy("created_at", firestore.Desc).
		Limit(limit)

	if cursor != "" {
		decoded, err := base64.StdEncoding.DecodeString(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		cursorTime, err := time.Parse(time.RFC3339Nano, string(decoded))
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		q = q.StartAfter(cursorTime)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var feedbacks []*model.Feedback
	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			logger.Error("fail to get feedback data from Firestore", zap.Error(err))
			return nil, fmt.Errorf("failed to get feedback data from Firestore: %w", err)
		}
		var doc feedbackDocument
		if err := docSnap.DataTo(&doc); err != nil {
			logger.Error("failed to deserialize feedback document", zap.Error(err))
			return nil, fmt.Errorf("failed to deserialize feedback document: %w", err)
		}
		feedbacks = append(feedbacks, toFeedbackModel(&doc))
	}

	return feedbacks, nil
}

func (r *FeedbackFirestoreRepository) GetFeedbacksByReviewerID(ctx context.Context, reviewerID string, limit int, cursor string) ([]*model.Feedback, error) {
	q := r.client.Collection(feedbacksCollection).
		Where("reviewer_id", "==", reviewerID).
		OrderBy("created_at", firestore.Desc).
		Limit(limit)

	if cursor != "" {
		decoded, err := base64.StdEncoding.DecodeString(cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		cursorTime, err := time.Parse(time.RFC3339Nano, string(decoded))
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		q = q.StartAfter(cursorTime)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var feedbacks []*model.Feedback
	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			logger.Error("fail to get feedback data from Firestore", zap.Error(err))
			return nil, fmt.Errorf("failed to get feedback data from Firestore: %w", err)
		}
		var doc feedbackDocument
		if err := docSnap.DataTo(&doc); err != nil {
			logger.Error("failed to deserialize feedback document", zap.Error(err))
			return nil, fmt.Errorf("failed to deserialize feedback document: %w", err)
		}
		feedbacks = append(feedbacks, toFeedbackModel(&doc))
	}

	return feedbacks, nil
}

func (r *FeedbackFirestoreRepository) GetFeedback(ctx context.Context, reviewerID, revieweeID, period string) (*model.Feedback, error) {
	iter := r.client.Collection(feedbacksCollection).
		Where("reviewer_id", "==", reviewerID).
		Where("reviewee_id", "==", revieweeID).
		Where("period", "==", period).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	docSnap, err := iter.Next()
	if err != nil {
		// No document found — not an error, just no match
		return nil, nil
	}

	var doc feedbackDocument
	if err := docSnap.DataTo(&doc); err != nil {
		logger.Error("failed to deserialize feedback document", zap.Error(err))
		return nil, fmt.Errorf("failed to deserialize feedback document: %w", err)
	}

	return toFeedbackModel(&doc), nil
}
