package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsongpon/delphi/internal/model"
	"google.golang.org/api/option"
)

// newTestClient creates a Firestore client connected to the emulator.
// It registers a cleanup function that deletes all documents in the users collection.
func newTestClient(t *testing.T) *firestore.Client {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST not set, skipping integration test")
	}

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "test-project", option.WithoutAuthentication())
	require.NoError(t, err)

	t.Cleanup(func() {
		iter := client.Collection(usersCollection).Documents(ctx)
		for {
			doc, err := iter.Next()
			if err != nil {
				break
			}
			_, _ = doc.Ref.Delete(ctx)
		}
		client.Close()
	})

	return client
}

// seedUser inserts a user document directly into Firestore for test setup.
func seedUser(t *testing.T, client *firestore.Client, user *model.User) {
	t.Helper()
	ctx := context.Background()
	doc := toDocument(user)
	_, err := client.Collection(usersCollection).Doc(user.ID).Set(ctx, doc)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// CreateUser tests
// ---------------------------------------------------------------------------

func TestCreateUser(t *testing.T) {
	client := newTestClient(t)
	repo := NewUserFirestoreRepository(client)
	ctx := context.Background()

	user := &model.User{
		ID:        "create-test-user",
		Name:      "John Doe",
		Email:     "john@example.com",
		Password:  "hashed-password",
		Title:     "Engineer",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	result, err := repo.CreateUser(ctx, user)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "create-test-user", result.ID)

	// Verify document exists in Firestore
	doc, err := client.Collection(usersCollection).Doc(result.ID).Get(ctx)
	require.NoError(t, err)

	var stored userDocument
	err = doc.DataTo(&stored)
	require.NoError(t, err)

	assert.Equal(t, "John Doe", stored.Name)
	assert.Equal(t, "john@example.com", stored.Email)
	assert.Equal(t, result.ID, stored.ID)
}

// ---------------------------------------------------------------------------
// GetUserByID tests
// ---------------------------------------------------------------------------

func TestGetUserByID_Success(t *testing.T) {
	client := newTestClient(t)
	repo := NewUserFirestoreRepository(client)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)
	seedUser(t, client, &model.User{
		ID:        "user-abc",
		Name:      "Alice",
		Email:     "alice@example.com",
		Password:  "hashed",
		Title:     "Engineer",
		TeamID:    "team-1",
		CreatedAt: now,
		UpdatedAt: now,
	})

	result, err := repo.GetUserByID(ctx, "user-abc")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "user-abc", result.ID)
	assert.Equal(t, "Alice", result.Name)
	assert.Equal(t, "alice@example.com", result.Email)
	assert.Equal(t, "Engineer", result.Title)
	assert.Equal(t, "team-1", result.TeamID)
}

func TestGetUserByID_NotFound(t *testing.T) {
	client := newTestClient(t)
	repo := NewUserFirestoreRepository(client)
	ctx := context.Background()

	result, err := repo.GetUserByID(ctx, "nonexistent-id")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "user not found")
}

// ---------------------------------------------------------------------------
// GetUsersByTeamID tests
// ---------------------------------------------------------------------------

func TestGetUsersByTeamID_Success(t *testing.T) {
	client := newTestClient(t)
	repo := NewUserFirestoreRepository(client)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)

	// Seed 2 users in team-alpha, 1 in team-beta
	seedUser(t, client, &model.User{ID: "u1", Name: "Alice", Email: "a@e.com", TeamID: "team-alpha", CreatedAt: now, UpdatedAt: now})
	seedUser(t, client, &model.User{ID: "u2", Name: "Bob", Email: "b@e.com", TeamID: "team-alpha", CreatedAt: now, UpdatedAt: now})
	seedUser(t, client, &model.User{ID: "u3", Name: "Charlie", Email: "c@e.com", TeamID: "team-beta", CreatedAt: now, UpdatedAt: now})

	users, err := repo.GetUsersByTeamID(ctx, "team-alpha")
	require.NoError(t, err)
	assert.Len(t, users, 2)

	for _, u := range users {
		assert.Equal(t, "team-alpha", u.TeamID)
	}
}

func TestGetUsersByTeamID_NoMatches(t *testing.T) {
	client := newTestClient(t)
	repo := NewUserFirestoreRepository(client)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)
	seedUser(t, client, &model.User{ID: "u1", Name: "Alice", Email: "a@e.com", TeamID: "team-alpha", CreatedAt: now, UpdatedAt: now})

	users, err := repo.GetUsersByTeamID(ctx, "team-nonexistent")
	require.NoError(t, err)
	assert.Empty(t, users)
}

// ---------------------------------------------------------------------------
// GetUserByEmail tests
// ---------------------------------------------------------------------------

func TestGetUserByEmail_Success(t *testing.T) {
	client := newTestClient(t)
	repo := NewUserFirestoreRepository(client)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)
	seedUser(t, client, &model.User{
		ID:        "user-email-test",
		Name:      "Diana",
		Email:     "diana@example.com",
		Password:  "hashed-pw",
		Title:     "Manager",
		TeamID:    "team-2",
		CreatedAt: now,
		UpdatedAt: now,
	})

	result, err := repo.GetUserByEmail(ctx, "diana@example.com")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "user-email-test", result.ID)
	assert.Equal(t, "Diana", result.Name)
	assert.Equal(t, "diana@example.com", result.Email)
	assert.Equal(t, "Manager", result.Title)
	assert.Equal(t, "team-2", result.TeamID)
}
