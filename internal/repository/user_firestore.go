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

const usersCollection = "users"

// Compile-time check that UserFirestoreRepository implements service.UserRepository.
var _ service.UserRepository = (*UserFirestoreRepository)(nil)

// userDocument is a Firestore-specific DTO for the User entity.
// It keeps Firestore struct tags out of the domain model.
type userDocument struct {
	ID        string    `firestore:"id"`
	Name      string    `firestore:"name"`
	Email     string    `firestore:"email"`
	Password  string    `firestore:"password"`
	Title     string    `firestore:"title"`
	CreatedAt time.Time `firestore:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at"`
}

func toDocument(user *model.User) *userDocument {
	return &userDocument{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Password:  user.Password,
		Title:     user.Title,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func toModel(doc *userDocument) *model.User {
	return &model.User{
		ID:        doc.ID,
		Name:      doc.Name,
		Email:     doc.Email,
		Password:  doc.Password,
		Title:     doc.Title,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}
}

// UserFirestoreRepository is a Firestore-backed implementation of service.UserRepository.
type UserFirestoreRepository struct {
	client *firestore.Client
}

// NewUserFirestoreRepository creates a new UserFirestoreRepository.
// The caller is responsible for closing the firestore.Client.
func NewUserFirestoreRepository(client *firestore.Client) *UserFirestoreRepository {
	return &UserFirestoreRepository{
		client: client,
	}
}

// CreateUser saves a user to the Firestore "users" collection.
// The caller is responsible for setting the user ID and timestamps before calling this method.
func (r *UserFirestoreRepository) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {

	doc := toDocument(user)

	docRef := r.client.Collection(usersCollection).Doc(doc.ID)
	_, err := docRef.Set(ctx, doc)
	if err != nil {
		logger.Error("failed to create user in firestore", zap.Error(err))
		return nil, fmt.Errorf("failed to create user in firestore: %w", err)
	}

	return user, nil
}

// GetUserByEmail queries the Firestore "users" collection for a user with the given email.
func (r *UserFirestoreRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	iter := r.client.Collection(usersCollection).Where("email", "==", email).Limit(1).Documents(ctx)
	defer iter.Stop()

	docSnap, err := iter.Next()
	if err != nil {
		logger.Error("failed to get user by email", zap.Error(err))
		return nil, fmt.Errorf("user not found: %w", err)
	}

	var doc userDocument
	if err := docSnap.DataTo(&doc); err != nil {
		logger.Error("failed to deserialize user document", zap.Error(err))
		return nil, fmt.Errorf("failed to deserialize user document: %w", err)
	}

	return toModel(&doc), nil
}
