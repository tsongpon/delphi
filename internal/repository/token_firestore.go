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
)

const passwordResetTokensCollection = "password_reset_tokens"

// Compile-time check that TokenFirestoreRepository implements service.TokenRepository.
var _ service.TokenRepository = (*TokenFirestoreRepository)(nil)

type tokenDocument struct {
	TokenHash string `firestore:"token_hash"`
	UserID    string `firestore:"user_id"`
	ExpiresAt int64  `firestore:"expires_at"` // Unix timestamp
	CreatedAt int64  `firestore:"created_at"` // Unix timestamp
}

type TokenFirestoreRepository struct {
	client *firestore.Client
}

func NewTokenFirestoreRepository(client *firestore.Client) *TokenFirestoreRepository {
	return &TokenFirestoreRepository{client: client}
}

func (r *TokenFirestoreRepository) SaveToken(ctx context.Context, token *model.PasswordResetToken) error {
	doc := &tokenDocument{
		TokenHash: token.TokenHash,
		UserID:    token.UserID,
		ExpiresAt: token.ExpiresAt.Unix(),
		CreatedAt: token.CreatedAt.Unix(),
	}
	_, err := r.client.Collection(passwordResetTokensCollection).Doc(token.TokenHash).Set(ctx, doc)
	if err != nil {
		logger.Error("failed to save password reset token", zap.Error(err))
		return fmt.Errorf("failed to save token: %w", err)
	}
	return nil
}

func (r *TokenFirestoreRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordResetToken, error) {
	iter := r.client.Collection(passwordResetTokensCollection).
		Where("token_hash", "==", tokenHash).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	docSnap, err := iter.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		logger.Error("failed to get password reset token", zap.Error(err))
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var doc tokenDocument
	if err := docSnap.DataTo(&doc); err != nil {
		return nil, fmt.Errorf("failed to deserialize token document: %w", err)
	}

	return &model.PasswordResetToken{
		TokenHash: doc.TokenHash,
		UserID:    doc.UserID,
		ExpiresAt: toTime(doc.ExpiresAt),
		CreatedAt: toTime(doc.CreatedAt),
	}, nil
}

func toTime(unix int64) time.Time {
	return time.Unix(unix, 0).UTC()
}

func (r *TokenFirestoreRepository) DeleteToken(ctx context.Context, tokenHash string) error {
	_, err := r.client.Collection(passwordResetTokensCollection).Doc(tokenHash).Delete(ctx)
	if err != nil {
		logger.Error("failed to delete password reset token", zap.Error(err))
		return fmt.Errorf("failed to delete token: %w", err)
	}
	return nil
}
