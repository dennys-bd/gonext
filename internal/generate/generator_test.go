package generate

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGenerators_IsWireOnly(t *testing.T) {
	gens := Generators("/some/root")
	if len(gens) != 1 {
		t.Fatalf("Generators: len = %d, want 1", len(gens))
	}
	if gens[0].Name != "wire" {
		t.Errorf("Generators[0].Name = %q, want %q", gens[0].Name, "wire")
	}
}

func TestRunAll(t *testing.T) {
	tests := []struct {
		name     string
		check    bool
		wantArgv []string
	}{
		{name: "run", check: false, wantArgv: []string{"go", "tool", "wire", "./backend/..."}},
		{name: "check", check: true, wantArgv: []string{"go", "tool", "wire", "diff", "./backend/..."}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotDir string
			var gotArgv []string
			calls := 0
			restore := stubRunFunc(func(_ context.Context, dir, name string, args ...string) error {
				calls++
				gotDir = dir
				gotArgv = append([]string{name}, args...)
				return nil
			})
			defer restore()

			if err := RunAll(context.Background(), "/some/root", tt.check); err != nil {
				t.Fatalf("RunAll: unexpected error: %v", err)
			}
			if calls != 1 {
				t.Fatalf("RunAll: calls = %d, want 1", calls)
			}
			if gotDir != "/some/root" {
				t.Errorf("RunAll: dir = %q, want %q", gotDir, "/some/root")
			}
			if strings.Join(gotArgv, " ") != strings.Join(tt.wantArgv, " ") {
				t.Errorf("RunAll: argv = %v, want %v", gotArgv, tt.wantArgv)
			}
		})
	}
}

func TestRunAll_CheckFailureReportsWireStaleMessage(t *testing.T) {
	wantErr := errors.New("exit status 1")
	restore := stubRunFunc(func(context.Context, string, string, ...string) error {
		return wantErr
	})
	defer restore()

	err := RunAll(context.Background(), "/some/root", true)
	if err == nil {
		t.Fatal("RunAll: expected error, got nil")
	}
	if !strings.HasPrefix(err.Error(), "wire_gen.go is stale (run gonext generate)") || !errors.Is(err, wantErr) {
		t.Errorf("RunAll: err = %v, want wire's stale message wrapping %v", err, wantErr)
	}
}

func TestRunGenerators_SkipsStepsWithNilCheck(t *testing.T) {
	gens := []Generator{{Name: "no-check", Run: []string{"true"}}}

	calls := 0
	restore := stubRunFunc(func(context.Context, string, string, ...string) error {
		calls++
		return nil
	})
	defer restore()

	if err := runGenerators(context.Background(), "/some/root", gens, true); err != nil {
		t.Fatalf("runGenerators: unexpected error: %v", err)
	}
	if calls != 0 {
		t.Errorf("runGenerators: calls = %d, want 0 (step has nil Check)", calls)
	}
}

func TestRunGenerators_StopsOnFirstFailure(t *testing.T) {
	gens := []Generator{
		{Name: "first", Run: []string{"first"}},
		{Name: "second", Run: []string{"second"}},
	}

	wantErr := errors.New("boom")
	calls := 0
	restore := stubRunFunc(func(_ context.Context, _ string, name string, _ ...string) error {
		calls++
		if name == "first" {
			return wantErr
		}
		return nil
	})
	defer restore()

	err := runGenerators(context.Background(), "/some/root", gens, false)
	if err == nil {
		t.Fatal("runGenerators: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "first") {
		t.Errorf("runGenerators: err = %v, want it to name the failing step %q", err, "first")
	}
	if calls != 1 {
		t.Errorf("runGenerators: calls = %d, want 1 (should stop after first failure)", calls)
	}
}

func stubRunFunc(fn func(ctx context.Context, dir, name string, args ...string) error) func() {
	orig := runFunc
	runFunc = fn
	return func() { runFunc = orig }
}
