package auth_test

import (
	"context"
	"testing"

	"github.com/dennys-bd/gonext/auth"
)

func TestIdentityFrom_RoundTrip(t *testing.T) {
	want := auth.Identity{UserID: "u-1", Role: "admin", Permissions: []string{"posts:delete"}}

	got, ok := auth.IdentityFrom(auth.WithIdentity(context.Background(), want))
	if !ok {
		t.Fatal("expected an identity to be present")
	}
	if got.UserID != want.UserID || got.Role != want.Role {
		t.Errorf("IdentityFrom() = %+v, want %+v", got, want)
	}
}

func TestIdentityFrom_Absent(t *testing.T) {
	if _, ok := auth.IdentityFrom(context.Background()); ok {
		t.Error("expected no identity in a bare context")
	}
}

// The middleware lives in another module and injects through its
// transport's own helper, so it writes ContextKey directly. This
// asserts the exported key is the one IdentityFrom reads.
func TestIdentityFrom_ReadsExportedKey(t *testing.T) {
	want := auth.Identity{UserID: "u-2"}
	ctx := context.WithValue(context.Background(), auth.ContextKey, want)

	got, ok := auth.IdentityFrom(ctx)
	if !ok || got.UserID != want.UserID {
		t.Errorf("IdentityFrom() = %+v, %v; want %+v, true", got, ok, want)
	}
}

func TestMustIdentity_PanicsWhenAbsent(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected MustIdentity to panic on a bare context")
		}
	}()

	auth.MustIdentity(context.Background())
}
