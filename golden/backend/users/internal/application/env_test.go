package application_test

import (
	"testing"

	"golden-app/backend/users/internal/application"
)

// One predicate decides every environment-gated behaviour in this
// domain — token exposure, the unconfirmed-login gate, and (via the
// presentation layer) the cookie's Secure flag. stg sits on the
// restricted side with prod.
func TestIsRelaxedEnv(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"dev", true},
		{"test", true},
		{"stg", false},
		{"prod", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			if got := application.IsRelaxedEnv(tt.env); got != tt.want {
				t.Fatalf("env %q: expected %v, got %v", tt.env, tt.want, got)
			}
		})
	}
}
