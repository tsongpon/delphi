package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/tsongpon/delphi/internal/model"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidResetToken = errors.New("invalid or expired reset token")
	ErrUserNotFound      = errors.New("user not found")
)

const resetTokenExpiry = time.Hour

// PasswordResetServiceImpl handles admin-initiated password reset links.
type PasswordResetServiceImpl struct {
	tokenRepo TokenRepository
	userRepo  UserRepository
	baseURL   string
}

func NewPasswordResetService(tokenRepo TokenRepository, userRepo UserRepository, baseURL string) *PasswordResetServiceImpl {
	return &PasswordResetServiceImpl{tokenRepo: tokenRepo, userRepo: userRepo, baseURL: baseURL}
}

// GenerateResetLink creates a one-time reset token for the given userID and returns the reset URL.
func (s *PasswordResetServiceImpl) GenerateResetLink(ctx context.Context, userID string) (string, time.Time, error) {
	if _, err := s.userRepo.GetUserByID(ctx, userID); err != nil {
		return "", time.Time{}, ErrUserNotFound
	}

	rawToken, err := generateSecureToken()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate token: %w", err)
	}

	tokenHash := hashToken(rawToken)
	now := time.Now()
	expiresAt := now.Add(resetTokenExpiry)

	if err := s.tokenRepo.SaveToken(ctx, &model.PasswordResetToken{
		TokenHash: tokenHash,
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to save token: %w", err)
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, rawToken)
	return resetLink, expiresAt, nil
}

// ResetPassword validates the token and updates the user's password.
func (s *PasswordResetServiceImpl) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	tokenHash := hashToken(rawToken)

	record, err := s.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to look up token: %w", err)
	}
	if record == nil || time.Now().After(record.ExpiresAt) {
		return ErrInvalidResetToken
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, record.UserID, string(hashed)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// One-time use — delete after success
	_ = s.tokenRepo.DeleteToken(ctx, tokenHash)

	return nil
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(h[:])
}
