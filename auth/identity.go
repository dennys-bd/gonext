// Package auth is gonext's published authentication contract, the
// one piece of a generated project that is imported rather than
// scaffolded. It depends only on the standard library.
package auth

// Identity is the minimal identity a validated credential resolves
// to. Permissions is prefetched for Role, not queried per
// HasPermission call.
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
