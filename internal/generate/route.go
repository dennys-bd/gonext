package generate

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// segmentKind classifies one route segment per the Next.js app router
// grammar.
type segmentKind int

const (
	segStatic segmentKind = iota
	segDynamic
	segCatchAll
	segOptionalCatchAll
	segGroup
)

// segment is one `/`-separated piece of a route.
type segment struct {
	raw   string
	kind  segmentKind
	param string
}

// parsedRoute is a route validated and classified by parseRoute.
type parsedRoute struct {
	raw      string
	segments []segment
}

// params returns the segments that can satisfy an operation's path
// parameter, in order: dynamic and catch-all segments only. A
// `[[...name]]` optional catch-all cannot satisfy one, so it is
// deliberately excluded here.
func (r parsedRoute) params() []segment {
	var out []segment
	for _, s := range r.segments {
		if s.kind == segDynamic || s.kind == segCatchAll {
			out = append(out, s)
		}
	}
	return out
}

// allParams returns every named segment of r — dynamic, catch-all and
// optional catch-all — in route order, for listing a route's full
// Params type even when an operation only uses some of them.
func (r parsedRoute) allParams() []segment {
	var out []segment
	for _, s := range r.segments {
		if s.kind == segDynamic || s.kind == segCatchAll || s.kind == segOptionalCatchAll {
			out = append(out, s)
		}
	}
	return out
}

// dir returns the destination directory for r under root.
func (r parsedRoute) dir(root string) string {
	return filepath.Join(root, "frontend", "app", filepath.FromSlash(r.raw))
}

var (
	optionalCatchAllRE = regexp.MustCompile(`^\[\[\.\.\.(.*)\]\]$`)
	catchAllRE         = regexp.MustCompile(`^\[\.\.\.(.*)\]$`)
	dynamicRE          = regexp.MustCompile(`^\[(.*)\]$`)
	paramNameRE        = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
)

// parseRoute validates raw against the Next.js app router's segment
// grammar and returns its classified segments. The destination
// directory is frontend/app/<raw> verbatim, one leading slash
// trimmed.
func parseRoute(raw string) (parsedRoute, error) {
	if raw == "" || strings.Contains(raw, `\`) {
		return parsedRoute{}, fmt.Errorf("invalid route %q", raw)
	}

	trimmed := strings.TrimPrefix(raw, "/")
	parts := strings.Split(trimmed, "/")
	segments := make([]segment, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		seg, err := parseSegment(raw, part, seen)
		if err != nil {
			return parsedRoute{}, err
		}
		segments = append(segments, seg)
	}
	return parsedRoute{raw: trimmed, segments: segments}, nil
}

// parseSegment classifies one path segment of route and validates
// its parameter name against seen, which it updates in place.
func parseSegment(route, part string, seen map[string]bool) (segment, error) {
	if m := optionalCatchAllRE.FindStringSubmatch(part); m != nil {
		name, err := claimParam(route, m[1], seen)
		if err != nil {
			return segment{}, err
		}
		return segment{raw: part, kind: segOptionalCatchAll, param: name}, nil
	}
	if m := catchAllRE.FindStringSubmatch(part); m != nil {
		name, err := claimParam(route, m[1], seen)
		if err != nil {
			return segment{}, err
		}
		return segment{raw: part, kind: segCatchAll, param: name}, nil
	}
	if m := dynamicRE.FindStringSubmatch(part); m != nil {
		name, err := claimParam(route, m[1], seen)
		if err != nil {
			return segment{}, err
		}
		return segment{raw: part, kind: segDynamic, param: name}, nil
	}
	if strings.HasPrefix(part, "(") || strings.HasPrefix(part, "@") {
		return segment{raw: part, kind: segGroup}, nil
	}
	if strings.HasPrefix(part, "_") {
		return segment{}, fmt.Errorf("invalid route %q: Next.js does not route pages under a private folder", route)
	}
	if part == "" || part == "." || part == ".." {
		return segment{}, fmt.Errorf("invalid route %q", route)
	}
	return segment{raw: part, kind: segStatic}, nil
}

// claimParam validates name against paramNameRE and records it in
// seen, rejecting a bad or already-used parameter name.
func claimParam(route, name string, seen map[string]bool) (string, error) {
	if !paramNameRE.MatchString(name) {
		return "", fmt.Errorf("invalid route %q: bad parameter name %q", route, name)
	}
	if seen[name] {
		return "", fmt.Errorf("invalid route %q: duplicate parameter %q", route, name)
	}
	seen[name] = true
	return name, nil
}
