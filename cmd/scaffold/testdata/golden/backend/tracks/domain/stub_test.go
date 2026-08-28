package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewStub_Success(t *testing.T) {
	now := time.Now().UTC()
	s, err := NewStub("id-1", "demo", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ID != "id-1" || s.Name != "demo" || !s.CreatedAt.Equal(now) {
		t.Fatalf("unexpected stub: %+v", s)
	}
}

func TestNewStub_EmptyName(t *testing.T) {
	_, err := NewStub("id-1", "", time.Now().UTC())
	if !errors.Is(err, ErrStubNameRequired) {
		t.Fatalf("expected ErrStubNameRequired, got %v", err)
	}
}
