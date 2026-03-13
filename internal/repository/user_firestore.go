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
	Role      string    `firestore:"role"`
	CreatedAt time.Time `firestore:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at"`
	TeamID    string    `firestore:"team_id"`
}

func toDocument(user *model.User) *userDocument {
	return &userDocument{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Password:  user.Password,
		Title:     user.Title,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		TeamID:    user.TeamID,
	}
}

func toModel(doc *userDocument) *model.User {
	return &model.User{
		ID:        doc.ID,
		Name:      doc.Name,
		Email:     doc.Email,
		Password:  doc.Password,
		Title:     doc.Title,
		Role:      doc.Role,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
		TeamID:    doc.TeamID,
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

// GetUserByID fetches a user by document ID from the Firestore "users" collection.
func (r *UserFirestoreRepository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	docSnap, err := r.client.Collection(usersCollection).Doc(id).Get(ctx)
	if err != nil {
		logger.Error("failed to get user by ID", zap.String("id", id), zap.Error(err))
		return nil, fmt.Errorf("user not found: %w", err)
	}

	var doc userDocument
	if err := docSnap.DataTo(&doc); err != nil {
		logger.Error("failed to deserialize user document", zap.Error(err))
		return nil, fmt.Errorf("failed to deserialize user document: %w", err)
	}

	return toModel(&doc), nil
}

// GetUsersByTeamID queries the Firestore "users" collection for all users with the given team ID.
func (r *UserFirestoreRepository) GetUsersByTeamID(ctx context.Context, teamID string) ([]*model.User, error) {
	iter := r.client.Collection(usersCollection).Where("team_id", "==", teamID).Documents(ctx)
	defer iter.Stop()

	var users []*model.User
	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			logger.Error("failed to get users by team ID", zap.Error(err))
			return nil, fmt.Errorf("failed to get users by team ID: %w", err)
		}

		var doc userDocument
		if err := docSnap.DataTo(&doc); err != nil {
			logger.Error("failed to deserialize user document", zap.Error(err))
			continue
		}
		users = append(users, toModel(&doc))
	}

	return users, nil
}

// UpdateRole updates only the role field of a user document.
func (r *UserFirestoreRepository) UpdateRole(ctx context.Context, userID, role string) error {
	_, err := r.client.Collection(usersCollection).Doc(userID).Update(ctx, []firestore.Update{
		{Path: "role", Value: role},
	})
	if err != nil {
		logger.Error("failed to update user role", zap.String("user_id", userID), zap.Error(err))
		return fmt.Errorf("failed to update role: %w", err)
	}
	return nil
}

// UpdatePassword updates only the password field of a user document.
func (r *UserFirestoreRepository) UpdatePassword(ctx context.Context, userID, hashedPassword string) error {
	_, err := r.client.Collection(usersCollection).Doc(userID).Update(ctx, []firestore.Update{
		{Path: "password", Value: hashedPassword},
	})
	if err != nil {
		logger.Error("failed to update user password", zap.String("user_id", userID), zap.Error(err))
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
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
