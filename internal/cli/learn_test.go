package cli

import (
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/curriculum"
	"github.com/gustavfredrikson/cka-dojo/internal/lab"
	"github.com/gustavfredrikson/cka-dojo/internal/progress"
)

func TestSummarizeModuleSeparatesAttemptsPassesAndMastery(t *testing.T) {
	module := &curriculum.Module{Labs: []*lab.Lab{
		{ID: "new"},
		{ID: "attempted"},
		{ID: "passed"},
		{ID: "mastered"},
	}}
	history := &progress.File{Labs: map[string]*progress.Attempt{
		"attempted": {Attempts: 1},
		"passed":    {Attempts: 1, Passes: 1},
		"mastered":  {Attempts: 2, Passes: 2, LastPassHints: 0},
	}}

	got := summarizeModule(module, history)
	if got != (moduleProgress{Attempted: 3, Passed: 2, Mastered: 1, Total: 4}) {
		t.Fatalf("summary = %+v", got)
	}
}

func TestNextExerciseUsesFirstUnlockedUnpassedStage(t *testing.T) {
	first := &lab.Lab{ID: "follow"}
	second := &lab.Lab{ID: "build", Prerequisites: []string{"follow"}}
	third := &lab.Lab{ID: "inspect", Prerequisites: []string{"build"}}
	module := &curriculum.Module{Labs: []*lab.Lab{first, second, third}}
	history := &progress.File{Labs: map[string]*progress.Attempt{
		"follow": {Attempts: 1, Passes: 1},
	}}

	if got := nextExercise(module, history); got != second {
		t.Fatalf("next = %#v, want build", got)
	}
	history.Labs["build"] = &progress.Attempt{Attempts: 1, Passes: 1}
	history.Labs["inspect"] = &progress.Attempt{Attempts: 1, Passes: 1}
	if got := nextExercise(module, history); got != nil {
		t.Fatalf("completed path still recommends %#v", got)
	}
}

func TestProgressFractionMarksCompletePasses(t *testing.T) {
	if got := progressFraction(5, 5, "✓"); got != "✓ 5/5" {
		t.Fatalf("complete fraction = %q", got)
	}
	if got := progressFraction(2, 5, "✓"); got != "2/5" {
		t.Fatalf("partial fraction = %q", got)
	}
}
