package application_test

import (
	"testing"

	"[PROJECT-NAME]/backend/users/internal/application"
)

// stg sits on the restricted side with prod, not the relaxed side with dev/test.
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
