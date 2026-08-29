package domain_test

import (
	"testing"
	"time"

	"[PROJECT-NAME]/backend/users/domain"
)

func TestToken_Usable(t *testing.T) {
	now := time.Now().UTC()
	usedAt := now.Add(-time.Minute)

	tests := []struct {
		name  string
		token domain.Token
		kind  domain.TokenKind
		want  bool
	}{
		{
			name:  "fresh token of the expected kind",
			token: domain.Token{Kind: domain.TokenKindEmailConfirmation, ExpiresAt: now.Add(time.Hour)},
			kind:  domain.TokenKindEmailConfirmation,
			want:  true,
		},
		{
			name:  "wrong kind",
			token: domain.Token{Kind: domain.TokenKindPasswordReset, ExpiresAt: now.Add(time.Hour)},
			kind:  domain.TokenKindEmailConfirmation,
			want:  false,
		},
		{
			name:  "expired",
			token: domain.Token{Kind: domain.TokenKindEmailConfirmation, ExpiresAt: now.Add(-time.Hour)},
			kind:  domain.TokenKindEmailConfirmation,
			want:  false,
		},
		{
			name:  "already consumed",
			token: domain.Token{Kind: domain.TokenKindEmailConfirmation, ExpiresAt: now.Add(time.Hour), UsedAt: &usedAt},
			kind:  domain.TokenKindEmailConfirmation,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.token.Usable(tt.kind, now); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
