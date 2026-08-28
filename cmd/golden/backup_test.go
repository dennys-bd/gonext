package main

import "testing"

func TestPlanBackup(t *testing.T) {
	tests := []struct {
		name           string
		goldenExists   bool
		goldenOldExist bool
		want           backupAction
	}{
		{
			name:           "no golden yet",
			goldenExists:   false,
			goldenOldExist: false,
			want:           actionGenerate,
		},
		{
			name:           "golden exists, no golden-old",
			goldenExists:   true,
			goldenOldExist: false,
			want:           actionBackupThenGenerate,
		},
		{
			name:           "golden and golden-old both exist",
			goldenExists:   true,
			goldenOldExist: true,
			want:           actionNeedsPrompt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := planBackup(tt.goldenExists, tt.goldenOldExist)
			if got != tt.want {
				t.Errorf("planBackup(%v, %v) = %v, want %v", tt.goldenExists, tt.goldenOldExist, got, tt.want)
			}
		})
	}
}

func TestValidateBackupName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid name", input: "experiment", wantErr: false},
		{name: "valid name with digits and hyphens", input: "wip-2", wantErr: false},
		{name: "empty", input: "", wantErr: true},
		{name: "collides with golden", input: "golden", wantErr: true},
		{name: "collides with golden-old", input: "golden-old", wantErr: true},
		{name: "contains path separator", input: "a/b", wantErr: true},
		{name: "path traversal", input: "..", wantErr: true},
		{name: "uppercase rejected", input: "Experiment", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBackupName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBackupName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
