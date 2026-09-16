package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"golden-app/backend/example/domain"
	"golden-app/backend/example/internal/infrastructure/postgres"
	"golden-app/backend/internal/database/dbtest"
)

func testTag(id string, at time.Time) domain.Tag {
	return domain.Tag{
		ID:        id,
		CreatedAt: at,
		UpdatedAt: at,
	}
}

func sameTag(a, b domain.Tag) bool {
	return a.ID == b.ID &&
		a.CreatedAt.Equal(b.CreatedAt) &&
		a.UpdatedAt.Equal(b.UpdatedAt)
}

func TestTagRepository_CreateAndGet(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewTagRepository(db)
	ctx := context.Background()
	tag := testTag("tag-1", time.Now().UTC().Truncate(time.Microsecond))

	if err := repo.Create(ctx, tag); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get(ctx, tag.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !sameTag(got, tag) {
		t.Fatalf("expected %+v, got %+v", tag, got)
	}
}

func TestTagRepository_GetNotFound(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewTagRepository(db)

	_, err := repo.Get(context.Background(), "does-not-exist")
	if !errors.Is(err, domain.ErrTagNotFound) {
		t.Fatalf("expected ErrTagNotFound, got %v", err)
	}
}

func TestTagRepository_List(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewTagRepository(db)
	ctx := context.Background()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	older := testTag("tag-a", base)
	newer := testTag("tag-c", base.Add(time.Hour))
	newerLowerID := testTag("tag-b", base.Add(time.Hour))
	for _, tag := range []domain.Tag{older, newer, newerLowerID} {
		if err := repo.Create(ctx, tag); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	all, err := repo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 3 || !sameTag(all[0], newerLowerID) || !sameTag(all[1], newer) || !sameTag(all[2], older) {
		t.Fatalf("expected newest first then by id, got %+v", all)
	}

	page, err := repo.List(ctx, 1, 1)
	if err != nil {
		t.Fatalf("list page: %v", err)
	}
	if len(page) != 1 || !sameTag(page[0], newer) {
		t.Fatalf("expected [%+v], got %+v", newer, page)
	}

	past, err := repo.List(ctx, 10, 5)
	if err != nil {
		t.Fatalf("list past the end: %v", err)
	}
	if past == nil || len(past) != 0 {
		t.Fatalf("expected an empty, non-nil slice, got %+v", past)
	}
}

func TestTagRepository_Update(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewTagRepository(db)
	ctx := context.Background()
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tag := testTag("tag-1", at)
	if err := repo.Create(ctx, tag); err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := repo.Update(ctx, domain.Tag{
		ID:        tag.ID,
		UpdatedAt: at.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !updated.CreatedAt.Equal(at) || !updated.UpdatedAt.Equal(at.Add(time.Hour)) {
		t.Fatalf("expected CreatedAt kept and UpdatedAt replaced, got %+v", updated)
	}
}

func TestTagRepository_UpdateNotFound(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewTagRepository(db)

	_, err := repo.Update(context.Background(), testTag("does-not-exist", time.Now().UTC()))
	if !errors.Is(err, domain.ErrTagNotFound) {
		t.Fatalf("expected ErrTagNotFound, got %v", err)
	}
}

func TestTagRepository_Delete(t *testing.T) {
	db, _ := dbtest.New(t)
	repo := postgres.NewTagRepository(db)
	ctx := context.Background()
	tag := testTag("tag-1", time.Now().UTC().Truncate(time.Microsecond))
	if err := repo.Create(ctx, tag); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.Delete(ctx, tag.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := repo.Delete(ctx, tag.ID); !errors.Is(err, domain.ErrTagNotFound) {
		t.Fatalf("expected ErrTagNotFound after delete, got %v", err)
	}
}
