package repository

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/tsongpon/delphi/internal/model"
	"google.golang.org/api/option"
)

func TestCreateUser(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST not set, skipping integration test")
	}

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "test-project", option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("failed to create firestore client: %v", err)
	}
	defer client.Close()

	repo := NewUserFirestoreRepository(client)

	user := &model.User{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashed-password",
		Title:    "Engineer",
	}

	result, err := repo.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if result.ID == "" {
		t.Error("expected ID to be set")
	}

	if result.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if result.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}

	// Verify document exists in Firestore
	doc, err := client.Collection(usersCollection).Doc(result.ID).Get(ctx)
	if err != nil {
		t.Fatalf("failed to read back document: %v", err)
	}

	var stored model.User
	if err := doc.DataTo(&stored); err != nil {
		t.Fatalf("failed to deserialize document: %v", err)
	}

	if stored.Name != "John Doe" {
		t.Errorf("expected name 'John Doe', got '%s'", stored.Name)
	}

	if stored.Email != "john@example.com" {
		t.Errorf("expected email 'john@example.com', got '%s'", stored.Email)
	}

	if stored.ID != result.ID {
		t.Errorf("expected ID '%s', got '%s'", result.ID, stored.ID)
	}
}
