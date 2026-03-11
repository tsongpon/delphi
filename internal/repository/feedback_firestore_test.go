package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsongpon/delphi/internal/model"
)

func TestCreateFeedback_Firestore_Success(t *testing.T) {
	client := newTestClient(t)
	repo := NewFeedbackFirestoreRepository(client)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)
	feedback := &model.Feedback{
		ID:                 "feedback-1",
		Period:             "1-2026",
		ReviewerID:         "reviewer-1",
		RevieweeID:         "reviewee-1",
		CommunicationScore: 5,
		LeadershipScore:    4,
		TechnicalScore:     5,
		CollaborationScore: 4,
		DeliveryScore:      3,
		StrengthsComment:   "Great communicator",
		WeaknessesComment:  "Could improve delivery",
		Visibility:         "named",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	result, err := repo.CreateFeedback(ctx, feedback)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "feedback-1", result.ID)

	// Verify document exists in Firestore
	doc, err := client.Collection(feedbacksCollection).Doc("feedback-1").Get(ctx)
	require.NoError(t, err)

	var stored feedbackDocument
	err = doc.DataTo(&stored)
	require.NoError(t, err)

	assert.Equal(t, "1-2026", stored.Period)
	assert.Equal(t, "reviewer-1", stored.ReviewerID)
	assert.Equal(t, "reviewee-1", stored.RevieweeID)
	assert.Equal(t, 5, stored.CommunicationScore)
	assert.Equal(t, "named", stored.Visibility)
}

func TestGetFeedback_Found(t *testing.T) {
	client := newTestClient(t)
	repo := NewFeedbackFirestoreRepository(client)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)

	// Seed directly via Firestore client
	doc := toFeedbackDocument(&model.Feedback{
		ID:                 "fb-1",
		Period:             "1-2026",
		ReviewerID:         "reviewer-1",
		RevieweeID:         "reviewee-1",
		CommunicationScore: 5,
		Visibility:         "named",
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	_, err := client.Collection(feedbacksCollection).Doc("fb-1").Set(ctx, doc)
	require.NoError(t, err)

	result, err := repo.GetFeedback(ctx, "reviewer-1", "reviewee-1", "1-2026")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "fb-1", result.ID)
	assert.Equal(t, "1-2026", result.Period)
	assert.Equal(t, "reviewer-1", result.ReviewerID)
	assert.Equal(t, "reviewee-1", result.RevieweeID)
}

func TestGetFeedback_NotFound(t *testing.T) {
	client := newTestClient(t)
	repo := NewFeedbackFirestoreRepository(client)
	ctx := context.Background()

	result, err := repo.GetFeedback(ctx, "reviewer-1", "reviewee-1", "1-2026")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetFeedbacksByReviewerID_Found(t *testing.T) {
	client := newTestClient(t)
	repo := NewFeedbackFirestoreRepository(client)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)

	doc := toFeedbackDocument(&model.Feedback{
		ID:                 "given-fb-1",
		Period:             "1-2026",
		ReviewerID:         "reviewer-1",
		RevieweeID:         "reviewee-1",
		CommunicationScore: 5,
		Visibility:         "named",
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	_, err := client.Collection(feedbacksCollection).Doc("given-fb-1").Set(ctx, doc)
	require.NoError(t, err)

	results, err := repo.GetFeedbacksByReviewerID(ctx, "reviewer-1", 10, "")
	require.NoError(t, err)
	require.Len(t, results, 1)

	assert.Equal(t, "given-fb-1", results[0].ID)
	assert.Equal(t, "reviewer-1", results[0].ReviewerID)
	assert.Equal(t, "reviewee-1", results[0].RevieweeID)
}

func TestGetFeedbacksByReviewerID_NotFound(t *testing.T) {
	client := newTestClient(t)
	repo := NewFeedbackFirestoreRepository(client)
	ctx := context.Background()

	results, err := repo.GetFeedbacksByReviewerID(ctx, "nonexistent-reviewer", 10, "")
	assert.NoError(t, err)
	assert.Empty(t, results)
}
