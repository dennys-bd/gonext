package auth

import "strings"

// SchemeName is the OpenAPI security scheme every rule references.
const SchemeName = "session"

// DefaultCookieName is the cookie the session credential travels in
// unless a project overrides it. Published so that the middleware
// reading the cookie and the handler setting it cannot disagree.
const DefaultCookieName = "session"

// Scope prefixes distinguishing a role requirement from a permission
// requirement inside a security requirement's scope list.
const (
	rolePrefix       = "role:"
	permissionPrefix = "perm:"
)

// Required declares that any valid credential is enough.
func Required() []map[string][]string {
	return []map[string][]string{{SchemeName: {}}}
}

// RequireRole declares that the caller's identity must have exactly
// role.
func RequireRole(role string) []map[string][]string {
	return []map[string][]string{{SchemeName: {rolePrefix + role}}}
}

// RequirePermission declares that the caller's identity must hold
// permission key.
func RequirePermission(key string) []map[string][]string {
	return []map[string][]string{{SchemeName: {permissionPrefix + key}}}
}

// Optional declares that an identity should be resolved when a
// credential is present, and that its absence is not an error. The
// empty first alternative is OpenAPI's own way of saying "no
// credential is also acceptable", so the published document stays
// standards-correct rather than carrying an invented marker.
func Optional() []map[string][]string {
	return []map[string][]string{{}, {SchemeName: {}}}
}

// Rule is one operation's decoded requirement.
type Rule struct {
	// Enabled reports whether the operation participates in
	// authentication at all. When false the middleware never reads a
	// credential, so an undeclared operation costs no lookup.
	Enabled bool
	// Optional reports that a missing or invalid credential must not
	// be rejected.
	Optional bool
	// Roles and Permissions are ANDed: every entry must be satisfied.
	Roles       []string
	Permissions []string
}

// RuleFor decodes an operation's OpenAPI security requirements.
//
// Each helper above produces exactly one rule; composing two helpers
// on one operation is not supported, and RuleFor simply flattens
// whatever non-empty alternatives it finds.
func RuleFor(security []map[string][]string) Rule {
	if len(security) == 0 {
		return Rule{}
	}

	rule := Rule{Enabled: true}
	for _, requirement := range security {
		if len(requirement) == 0 {
			rule.Optional = true
			continue
		}
		for _, scope := range requirement[SchemeName] {
			switch {
			case strings.HasPrefix(scope, rolePrefix):
				rule.Roles = append(rule.Roles, strings.TrimPrefix(scope, rolePrefix))
			case strings.HasPrefix(scope, permissionPrefix):
				rule.Permissions = append(rule.Permissions, strings.TrimPrefix(scope, permissionPrefix))
			}
		}
	}
	return rule
}
