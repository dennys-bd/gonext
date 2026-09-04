package auth_test

import (
	"reflect"
	"testing"

	"github.com/dennys-bd/gonext/auth"
)

func TestRuleFor(t *testing.T) {
	tests := []struct {
		name     string
		security []map[string][]string
		want     auth.Rule
	}{
		{
			name:     "no declaration is not enabled",
			security: nil,
			want:     auth.Rule{},
		},
		{
			name:     "required",
			security: auth.Required(),
			want:     auth.Rule{Enabled: true},
		},
		{
			name:     "role",
			security: auth.RequireRole("admin"),
			want:     auth.Rule{Enabled: true, Roles: []string{"admin"}},
		},
		{
			name:     "permission",
			security: auth.RequirePermission("posts:delete"),
			want:     auth.Rule{Enabled: true, Permissions: []string{"posts:delete"}},
		},
		{
			name:     "optional",
			security: auth.Optional(),
			want:     auth.Rule{Enabled: true, Optional: true},
		},
		{
			name:     "an alternative for another scheme is ignored",
			security: []map[string][]string{{"oauth2": {"read"}}},
			want:     auth.Rule{Enabled: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auth.RuleFor(tt.security)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RuleFor() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// Optional must encode as OpenAPI's own idiom for optional auth: an
// empty requirement offered as an alternative to the real one. A
// generated client reads this, so the encoding is contract, not
// implementation detail.
func TestOptional_EncodesEmptyAlternative(t *testing.T) {
	got := auth.Optional()

	if len(got) != 2 {
		t.Fatalf("expected 2 alternatives, got %d: %+v", len(got), got)
	}
	if len(got[0]) != 0 {
		t.Errorf("expected the first alternative to be empty, got %+v", got[0])
	}
	if _, ok := got[1][auth.SchemeName]; !ok {
		t.Errorf("expected the second alternative to name %q, got %+v", auth.SchemeName, got[1])
	}
}
