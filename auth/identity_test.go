package auth_test

import (
	"testing"

	"github.com/dennys-bd/gonext/auth"
)

func TestIdentity_HasRole(t *testing.T) {
	id := auth.Identity{UserID: "u-1", Role: "admin"}

	if !id.HasRole("admin") {
		t.Error("expected HasRole(admin) to be true")
	}
	if id.HasRole("user") {
		t.Error("expected HasRole(user) to be false")
	}
}

func TestIdentity_HasPermission(t *testing.T) {
	id := auth.Identity{Role: "admin", Permissions: []string{"posts:delete", "posts:edit"}}

	if !id.HasPermission("posts:delete") {
		t.Error("expected HasPermission(posts:delete) to be true")
	}
	if id.HasPermission("posts:publish") {
		t.Error("expected HasPermission(posts:publish) to be false")
	}
}

func TestIdentity_HasPermission_NoPermissionsSeeded(t *testing.T) {
	id := auth.Identity{Role: "user"}

	if id.HasPermission("posts:delete") {
		t.Error("expected HasPermission to be false when no permissions are seeded")
	}
}
