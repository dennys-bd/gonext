package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"golden-app/backend/example/domain"
	"golden-app/backend/example/internal/infrastructure/memory"
)

func seedTag(t *testing.T, repo *memory.TagRepository, id string) domain.Tag {
	t.Helper()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tag := domain.Tag{
		ID:        id,
		CreatedAt: at,
		UpdatedAt: at,
	}
	if err := repo.Create(context.Background(), tag); err != nil {
		t.Fatalf("seeding a tag: %v", err)
	}
	return tag
}

func TestTagService_Create(t *testing.T) {
	repo := memory.NewTagRepository()
	svc := NewTagService(repo)
	ctx := context.Background()

	created, err := svc.CreateTag(ctx)
	if err != nil {
		t.Fatalf("create: unexpected error: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("unexpected tag: %+v", created)
	}
	if created.CreatedAt.IsZero() || !created.UpdatedAt.Equal(created.CreatedAt) {
		t.Fatalf("expected both timestamps set to the creation time, got %+v", created)
	}

	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: unexpected error: %v", err)
	}
	if got != created {
		t.Fatalf("expected %+v, got %+v", created, got)
	}
}

func TestTagService_GetNotFound(t *testing.T) {
	svc := NewTagService(memory.NewTagRepository())

	_, err := svc.GetTag(context.Background(), "does-not-exist")
	if !errors.Is(err, domain.ErrTagNotFound) {
		t.Fatalf("expected ErrTagNotFound, got %v", err)
	}
}

func TestTagService_List(t *testing.T) {
	repo := memory.NewTagRepository()
	svc := NewTagService(repo)
	seeded := seedTag(t, repo, "tag-1")

	tags, err := svc.ListTags(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("list: unexpected error: %v", err)
	}
	if len(tags) != 1 || tags[0] != seeded {
		t.Fatalf("expected [%+v], got %+v", seeded, tags)
	}
}

func TestTagService_Update(t *testing.T) {
	repo := memory.NewTagRepository()
	svc := NewTagService(repo)
	seeded := seedTag(t, repo, "tag-1")

	updated, err := svc.UpdateTag(context.Background(), seeded.ID)
	if err != nil {
		t.Fatalf("update: unexpected error: %v", err)
	}
	if updated.ID != seeded.ID {
		t.Fatalf("unexpected tag: %+v", updated)
	}
	if !updated.CreatedAt.Equal(seeded.CreatedAt) || !updated.UpdatedAt.After(seeded.UpdatedAt) {
		t.Fatalf("expected CreatedAt kept and UpdatedAt advanced, got %+v", updated)
	}
}

func TestTagService_UpdateNotFound(t *testing.T) {
	svc := NewTagService(memory.NewTagRepository())

	_, err := svc.UpdateTag(context.Background(), "does-not-exist")
	if !errors.Is(err, domain.ErrTagNotFound) {
		t.Fatalf("expected ErrTagNotFound, got %v", err)
	}
}

func TestTagService_Delete(t *testing.T) {
	repo := memory.NewTagRepository()
	svc := NewTagService(repo)
	seeded := seedTag(t, repo, "tag-1")
	ctx := context.Background()

	if err := svc.DeleteTag(ctx, seeded.ID); err != nil {
		t.Fatalf("delete: unexpected error: %v", err)
	}
	if err := svc.DeleteTag(ctx, seeded.ID); !errors.Is(err, domain.ErrTagNotFound) {
		t.Fatalf("expected ErrTagNotFound after delete, got %v", err)
	}
}
