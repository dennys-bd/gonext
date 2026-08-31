// Package auth is gonext's published authentication contract. It is
// the one piece of a generated project that is imported rather than
// scaffolded, so that a provider adapter written outside the project
// — Clerk, Supabase, Auth0 — has a stable type to compile against.
//
// It depends on the standard library only, and must stay that way:
// every generated backend takes it as a dependency.
package auth

// Identity is the minimal identity a validated credential resolves
// to — enough for the auth middleware to gate a route by role or
// permission without a second lookup. Permissions is prefetched for
// Role by the Resolver, not queried per HasPermission call.
type Identity struct {
	UserID      string
	Role        string
	Permissions []string
}

// HasRole reports whether the identity's role is exactly role.
func (i Identity) HasRole(role string) bool {
	return i.Role == role
}

// HasPermission reports whether key is among the permissions
// prefetched for the identity's role. A project that never seeds its
// role/permission tables simply gets false for every key.
func (i Identity) HasPermission(key string) bool {
	for _, p := range i.Permissions {
		if p == key {
			return true
		}
	}
	return false
}
