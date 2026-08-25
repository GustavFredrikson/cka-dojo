package progress

import (
	"testing"
	"time"
)

func TestMasteryNeedsTwoPassesAndNoHints(t *testing.T) {
	for _, tc := range []struct {
		name string
		a    Attempt
		want bool
	}{
		{"one clean pass is not mastery", Attempt{Passes: 1, LastPassed: true}, false},
		{"two passes, last one clean", Attempt{Passes: 2, LastPassed: true}, true},
		{"two passes, last one hinted", Attempt{Passes: 2, LastPassed: true, LastHints: 1}, false},
		{"two passes, last attempt failed", Attempt{Passes: 2, LastPassed: false}, false},
	} {
		if got := tc.a.Mastered(); got != tc.want {
			t.Errorf("%s: Mastered = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestRegradingDoesNotInflatePasses is the reason grading takes a firstPass
// flag: a learner checking their work twice has not passed twice.
func TestRegradingDoesNotInflatePasses(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())
	f := &File{Version: 1, Labs: map[string]*Attempt{}}

	f.StartAttempt("lab-a", 42, "v1", []string{"services"}, []string{"troubleshooting"})
	f.RecordGrade("lab-a", true, true, 90*time.Second, 0, false)
	f.RecordGrade("lab-a", true, false, 120*time.Second, 0, false)

	a := f.Get("lab-a")
	if a.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", a.Attempts)
	}
	if a.Passes != 1 {
		t.Errorf("passes = %d, want 1", a.Passes)
	}
	if a.BestSeconds != 90 {
		t.Errorf("bestSeconds = %d, want 90", a.BestSeconds)
	}
}

func TestHintedPassIsNotClean(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())
	f := &File{Version: 1, Labs: map[string]*Attempt{}}
	f.StartAttempt("lab-a", 1, "", nil, nil)
	f.RecordGrade("lab-a", true, true, time.Minute, 2, false)
	if got := f.Get("lab-a").CleanPasses; got != 0 {
		t.Errorf("cleanPasses = %d, want 0", got)
	}
}

func TestBySkillShowsUnattemptedSkillsAsGaps(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())
	f := &File{Version: 1, Labs: map[string]*Attempt{}}
	f.StartAttempt("lab-a", 1, "", []string{"services"}, nil)
	f.RecordGrade("lab-a", true, true, time.Minute, 0, false)

	stats := f.BySkill(map[string][]string{
		"lab-a": {"services"},
		"lab-b": {"storage"}, // never attempted
	})
	byName := map[string]SkillStat{}
	for _, s := range stats {
		byName[s.Skill] = s
	}
	if byName["storage"].Attempts != 0 || byName["storage"].Labs != 1 {
		t.Errorf("untouched skill missing from report: %+v", byName["storage"])
	}
	if byName["services"].Passes != 1 {
		t.Errorf("services stat = %+v", byName["services"])
	}
}

func TestRoundTripsThroughDisk(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())
	f, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	f.StartAttempt("lab-a", 7, "v2", []string{"rbac"}, []string{"cluster-architecture"})
	if err := f.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	again, err := Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := again.Get("lab-a").LastSeed; got != 7 {
		t.Errorf("seed did not survive a round trip: %d", got)
	}
}
