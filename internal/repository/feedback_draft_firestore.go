package repository

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/tsongpon/delphi/internal/logger"
	"github.com/tsongpon/delphi/internal/model"
	"github.com/tsongpon/delphi/internal/service"
	"go.uber.org/zap"
)

const feedbackDraftsCollection = "feedback_drafts"

// Compile-time check that FeedbackDraftFirestoreRepository implements service.FeedbackDraftRepository.
var _ service.FeedbackDraftRepository = (*FeedbackDraftFirestoreRepository)(nil)

type feedbackDraftDocument struct {
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

func toFeedbackDraftDocument(d *model.FeedbackDraft) *feedbackDraftDocument {
	return &feedbackDraftDocument{
		ID:                 d.ID,
		Period:             d.Period,
		RevieweeID:         d.RevieweeID,
		ReviewerID:         d.ReviewerID,
		CommunicationScore: d.CommunicationScore,
		LeadershipScore:    d.LeadershipScore,
		TechnicalScore:     d.TechnicalScore,
		CollaborationScore: d.CollaborationScore,
		DeliveryScore:      d.DeliveryScore,
		StrengthsComment:   d.StrengthsComment,
		WeaknessesComment:  d.WeaknessesComment,
		Visibility:         d.Visibility,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}

func toFeedbackDraftModel(doc *feedbackDraftDocument) *model.FeedbackDraft {
	return &model.FeedbackDraft{
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

// draftDocID returns the deterministic composite document ID for a draft.
func draftDocID(reviewerID, revieweeID, period string) string {
	return reviewerID + "_" + revieweeID + "_" + period
}

type FeedbackDraftFirestoreRepository struct {
	client *firestore.Client
}

func NewFeedbackDraftFirestoreRepository(client *firestore.Client) *FeedbackDraftFirestoreRepository {
	return &FeedbackDraftFirestoreRepository{client: client}
}

func (r *FeedbackDraftFirestoreRepository) UpsertDraft(ctx context.Context, draft *model.FeedbackDraft) (*model.FeedbackDraft, error) {
	doc := toFeedbackDraftDocument(draft)
	docID := draftDocID(draft.ReviewerID, draft.RevieweeID, draft.Period)
	doc.ID = docID

	docRef := r.client.Collection(feedbackDraftsCollection).Doc(docID)
	_, err := docRef.Set(ctx, doc)
	if err != nil {
		logger.Error("failed to upsert feedback draft in firestore", zap.Error(err))
		return nil, fmt.Errorf("failed to upsert feedback draft in firestore: %w", err)
	}

	draft.ID = docID
	return draft, nil
}

func (r *FeedbackDraftFirestoreRepository) GetDraft(ctx context.Context, reviewerID, revieweeID, period string) (*model.FeedbackDraft, error) {
	docID := draftDocID(reviewerID, revieweeID, period)
	docSnap, err := r.client.Collection(feedbackDraftsCollection).Doc(docID).Get(ctx)
	if err != nil {
		// Document not found is not an error — return nil
		return nil, nil
	}

	var doc feedbackDraftDocument
	if err := docSnap.DataTo(&doc); err != nil {
		logger.Error("failed to deserialize feedback draft document", zap.Error(err))
		return nil, fmt.Errorf("failed to deserialize feedback draft document: %w", err)
	}

	return toFeedbackDraftModel(&doc), nil
}

func (r *FeedbackDraftFirestoreRepository) GetDraftsByReviewerID(ctx context.Context, reviewerID string) ([]*model.FeedbackDraft, error) {
	iter := r.client.Collection(feedbackDraftsCollection).
		Where("reviewer_id", "==", reviewerID).
		Documents(ctx)
	defer iter.Stop()

	var drafts []*model.FeedbackDraft
	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			logger.Error("failed to get feedback drafts from firestore", zap.Error(err))
			return nil, fmt.Errorf("failed to get feedback drafts from firestore: %w", err)
		}
		var doc feedbackDraftDocument
		if err := docSnap.DataTo(&doc); err != nil {
			logger.Error("failed to deserialize feedback draft document", zap.Error(err))
			return nil, fmt.Errorf("failed to deserialize feedback draft document: %w", err)
		}
		drafts = append(drafts, toFeedbackDraftModel(&doc))
	}

	return drafts, nil
}

func (r *FeedbackDraftFirestoreRepository) DeleteDraft(ctx context.Context, reviewerID, revieweeID, period string) error {
	docID := draftDocID(reviewerID, revieweeID, period)
	_, err := r.client.Collection(feedbackDraftsCollection).Doc(docID).Delete(ctx)
	if err != nil {
		logger.Error("failed to delete feedback draft from firestore", zap.Error(err))
		return fmt.Errorf("failed to delete feedback draft from firestore: %w", err)
	}
	return nil
}
