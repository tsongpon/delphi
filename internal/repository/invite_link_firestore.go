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

const inviteLinksCollection = "invite_links"

// Compile-time check that InviteLinkFirestoreRepository implements service.InviteLinkRepository.
var _ service.InviteLinkRepository = (*InviteLinkFirestoreRepository)(nil)

type inviteLinkDocument struct {
	ID        string `firestore:"id"`
	Token     string `firestore:"token"`
	TeamID    string `firestore:"team_id"`
	TeamName  string `firestore:"team_name"`
	CreatedBy string `firestore:"created_by"`
	Role      string `firestore:"role"`
	ExpiresAt int64  `firestore:"expires_at"` // Unix timestamp
	UsedCount int    `firestore:"used_count"`
	CreatedAt int64  `firestore:"created_at"` // Unix timestamp
}

func toInviteLinkDocument(link *model.InviteLink) *inviteLinkDocument {
	return &inviteLinkDocument{
		ID:        link.ID,
		Token:     link.Token,
		TeamID:    link.TeamID,
		TeamName:  link.TeamName,
		CreatedBy: link.CreatedBy,
		Role:      link.Role,
		ExpiresAt: link.ExpiresAt.Unix(),
		UsedCount: link.UsedCount,
		CreatedAt: link.CreatedAt.Unix(),
	}
}

func toInviteLinkModel(doc *inviteLinkDocument) *model.InviteLink {
	return &model.InviteLink{
		ID:        doc.ID,
		Token:     doc.Token,
		TeamID:    doc.TeamID,
		TeamName:  doc.TeamName,
		CreatedBy: doc.CreatedBy,
		Role:      doc.Role,
		ExpiresAt: time.Unix(doc.ExpiresAt, 0).UTC(),
		UsedCount: doc.UsedCount,
		CreatedAt: time.Unix(doc.CreatedAt, 0).UTC(),
	}
}

type InviteLinkFirestoreRepository struct {
	client *firestore.Client
}

func NewInviteLinkFirestoreRepository(client *firestore.Client) *InviteLinkFirestoreRepository {
	return &InviteLinkFirestoreRepository{client: client}
}

func (r *InviteLinkFirestoreRepository) CreateInviteLink(ctx context.Context, link *model.InviteLink) (*model.InviteLink, error) {
	doc := toInviteLinkDocument(link)
	_, err := r.client.Collection(inviteLinksCollection).Doc(doc.ID).Set(ctx, doc)
	if err != nil {
		logger.Error("failed to create invite link in firestore", zap.Error(err))
		return nil, fmt.Errorf("failed to create invite link: %w", err)
	}
	return toInviteLinkModel(doc), nil
}

func (r *InviteLinkFirestoreRepository) GetByID(ctx context.Context, id string) (*model.InviteLink, error) {
	docSnap, err := r.client.Collection(inviteLinksCollection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		logger.Error("failed to get invite link by id", zap.String("id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to get invite link: %w", err)
	}

	var doc inviteLinkDocument
	if err := docSnap.DataTo(&doc); err != nil {
		return nil, fmt.Errorf("failed to deserialize invite link document: %w", err)
	}
	return toInviteLinkModel(&doc), nil
}

func (r *InviteLinkFirestoreRepository) GetByTeamID(ctx context.Context, teamID string) ([]*model.InviteLink, error) {
	iter := r.client.Collection(inviteLinksCollection).
		Where("team_id", "==", teamID).
		Documents(ctx)
	defer iter.Stop()

	var links []*model.InviteLink
	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			logger.Error("failed to list invite links by team", zap.String("team_id", teamID), zap.Error(err))
			return nil, fmt.Errorf("failed to list invite links: %w", err)
		}
		var doc inviteLinkDocument
		if err := docSnap.DataTo(&doc); err != nil {
			return nil, fmt.Errorf("failed to deserialize invite link document: %w", err)
		}
		links = append(links, toInviteLinkModel(&doc))
	}
	return links, nil
}

func (r *InviteLinkFirestoreRepository) DeleteInviteLink(ctx context.Context, id string) error {
	_, err := r.client.Collection(inviteLinksCollection).Doc(id).Delete(ctx)
	if err != nil {
		logger.Error("failed to delete invite link", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("failed to delete invite link: %w", err)
	}
	return nil
}

func (r *InviteLinkFirestoreRepository) IncrementUsedCount(ctx context.Context, id string) error {
	_, err := r.client.Collection(inviteLinksCollection).Doc(id).Update(ctx, []firestore.Update{
		{Path: "used_count", Value: firestore.Increment(1)},
	})
	if err != nil {
		logger.Error("failed to increment invite link used count", zap.String("id", id), zap.Error(err))
		return fmt.Errorf("failed to increment used count: %w", err)
	}
	return nil
}
