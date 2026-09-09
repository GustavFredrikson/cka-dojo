package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDevModeSources covers both ways a run declares itself development, and
// the explicit off switch that has to beat the marker file for a learner who
// studies from a checkout.
func TestDevModeSources(t *testing.T) {
	for _, tc := range []struct {
		name   string
		env    string
		marker bool
		want   bool
	}{
		{"nothing set", "", false, false},
		{"env on", "1", false, true},
		{"env true", "TRUE", false, true},
		{"env off", "0", false, false},
		{"marker alone", "", true, true},
		{"env off beats the marker", "false", true, false},
		{"env on without a marker", "yes", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			t.Setenv("DOJO_DEV", tc.env)
			if tc.marker {
				if err := os.WriteFile(filepath.Join(dir, DevMarker), nil, 0o644); err != nil {
					t.Fatalf("write marker: %v", err)
				}
			}
			if got := DevMode(); got != tc.want {
				t.Errorf("DevMode = %v, want %v", got, tc.want)
			}
		})
	}
}
