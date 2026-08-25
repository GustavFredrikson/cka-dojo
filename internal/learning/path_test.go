package learning

import (
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/lab"
	"github.com/gustavfredrikson/cka-dojo/internal/progress"
)

func TestPassAndMasteryGatesAreDistinct(t *testing.T) {
	history := &progress.File{Labs: map[string]*progress.Attempt{
		"follow":  {Passes: 1},
		"inspect": {Passes: 1},
	}}
	exercise := &lab.Lab{
		Prerequisites:        []string{"follow"},
		MasteryPrerequisites: []string{"inspect"},
	}
	missing := Missing(exercise, history)
	if len(missing) != 1 || missing[0].LabID != "inspect" || !missing[0].Mastery {
		t.Fatalf("missing = %+v, want inspect mastery", missing)
	}
	history.Labs["inspect"] = &progress.Attempt{Passes: 2, LastPassHints: 0}
	if !Unlocked(exercise, history) {
		t.Error("exercise stayed locked after all gates were met")
	}
}

func TestStatusDistinguishesPassedMasteredAndLocked(t *testing.T) {
	history := &progress.File{Labs: map[string]*progress.Attempt{
		"passed":   {Passes: 1},
		"mastered": {Passes: 2, LastPassHints: 0},
	}}
	if got := Status(&lab.Lab{ID: "passed"}, history); got != "✓" {
		t.Errorf("passed status = %q", got)
	}
	if got := Status(&lab.Lab{ID: "mastered"}, history); got != "★" {
		t.Errorf("mastered status = %q", got)
	}
	if got := Status(&lab.Lab{ID: "locked", Prerequisites: []string{"missing"}}, history); got != "🔒" {
		t.Errorf("locked status = %q", got)
	}
}
