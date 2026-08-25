package cli

import (
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/progress"
)

func TestFormatSeconds(t *testing.T) {
	for _, tc := range []struct {
		seconds int
		want    string
	}{
		{0, "-"},
		{8, "0:08"},
		{90, "1:30"},
		{3601, "60:01"},
	} {
		if got := formatSeconds(tc.seconds); got != tc.want {
			t.Errorf("formatSeconds(%d) = %q, want %q", tc.seconds, got, tc.want)
		}
	}
}

func TestSeedOf(t *testing.T) {
	if got := seedOf(&progress.Attempt{}); got != "-" {
		t.Errorf("empty seed = %q, want -", got)
	}
	if got := seedOf(&progress.Attempt{LastSeed: 9182731}); got != "9182731" {
		t.Errorf("seed = %q, want 9182731", got)
	}
}
