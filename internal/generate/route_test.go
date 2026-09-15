package generate

import "testing"

func TestParseRoute(t *testing.T) {
	tests := []struct {
		name       string
		route      string
		wantRaw    string
		wantKinds  []segmentKind
		wantParams []string
		wantErr    string
	}{
		{
			name:       "static then dynamic",
			route:      "stubs/[id]",
			wantRaw:    "stubs/[id]",
			wantKinds:  []segmentKind{segStatic, segDynamic},
			wantParams: []string{"id"},
		},
		{
			name:       "leading slash trimmed",
			route:      "/stubs/[id]",
			wantRaw:    "stubs/[id]",
			wantKinds:  []segmentKind{segStatic, segDynamic},
			wantParams: []string{"id"},
		},
		{
			name:       "catch-all",
			route:      "docs/[...slug]",
			wantRaw:    "docs/[...slug]",
			wantKinds:  []segmentKind{segStatic, segCatchAll},
			wantParams: []string{"slug"},
		},
		{
			name:      "optional catch-all",
			route:     "docs/[[...slug]]",
			wantRaw:   "docs/[[...slug]]",
			wantKinds: []segmentKind{segStatic, segOptionalCatchAll},
			// params() excludes optional catch-alls: they can never
			// satisfy a required path parameter.
			wantParams: nil,
		},
		{
			name:       "group",
			route:      "(admin)/stubs/[id]",
			wantRaw:    "(admin)/stubs/[id]",
			wantKinds:  []segmentKind{segGroup, segStatic, segDynamic},
			wantParams: []string{"id"},
		},
		{
			name:      "intercepting single dot",
			route:     "(.)photo/[id]",
			wantRaw:   "(.)photo/[id]",
			wantKinds: []segmentKind{segGroup, segDynamic},
			// Note: "(.)photo" is one segment classified as a group
			// (it starts with "(").
			wantParams: []string{"id"},
		},
		{
			name:       "intercepting double-double",
			route:      "(..)(..)x",
			wantRaw:    "(..)(..)x",
			wantKinds:  []segmentKind{segGroup},
			wantParams: nil,
		},
		{
			name:       "slot",
			route:      "@modal/[id]",
			wantRaw:    "@modal/[id]",
			wantKinds:  []segmentKind{segGroup, segDynamic},
			wantParams: []string{"id"},
		},
		{
			name:    "private segment",
			route:   "_lib/x",
			wantErr: `invalid route "_lib/x": Next.js does not route pages under a private folder`,
		},
		{
			name:    "empty middle segment",
			route:   "a//b",
			wantErr: `invalid route "a//b"`,
		},
		{
			name:    "parent traversal segment",
			route:   "a/../b",
			wantErr: `invalid route "a/../b"`,
		},
		{
			name:    "backslash",
			route:   `a\b`,
			wantErr: `invalid route "a\\b"`,
		},
		{
			name:    "bad parameter name",
			route:   "[1bad]",
			wantErr: `invalid route "[1bad]": bad parameter name "1bad"`,
		},
		{
			name:    "duplicate parameter",
			route:   "[id]/[id]",
			wantErr: `invalid route "[id]/[id]": duplicate parameter "id"`,
		},
		{
			name:    "empty route",
			route:   "",
			wantErr: `invalid route ""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRoute(tt.route)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("parseRoute(%q): err = %v, want %q", tt.route, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRoute(%q): unexpected error: %v", tt.route, err)
			}
			if got.raw != tt.wantRaw {
				t.Errorf("parseRoute(%q): raw = %q, want %q", tt.route, got.raw, tt.wantRaw)
			}
			if len(got.segments) != len(tt.wantKinds) {
				t.Fatalf("parseRoute(%q): segments = %d, want %d", tt.route, len(got.segments), len(tt.wantKinds))
			}
			for i, wantKind := range tt.wantKinds {
				if got.segments[i].kind != wantKind {
					t.Errorf("parseRoute(%q): segments[%d].kind = %v, want %v", tt.route, i, got.segments[i].kind, wantKind)
				}
			}
			params := got.params()
			if len(params) != len(tt.wantParams) {
				t.Fatalf("parseRoute(%q): params = %v, want %v", tt.route, params, tt.wantParams)
			}
			for i, wantName := range tt.wantParams {
				if params[i].param != wantName {
					t.Errorf("parseRoute(%q): params[%d] = %q, want %q", tt.route, i, params[i].param, wantName)
				}
			}
		})
	}
}
