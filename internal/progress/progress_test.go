package progress

import (
	"os"
	"path/filepath"
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
		{"two passes, last successful one hinted", Attempt{Passes: 2, LastPassHints: 1}, false},
		{"later failed attempt does not revoke mastery", Attempt{Passes: 2, LastPassed: false}, true},
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
	if f.Get("lab-a").Mastered() {
		t.Error("a hinted latest pass counted toward mastery")
	}
}

func TestStartingAnotherAttemptDoesNotRevokeMastery(t *testing.T) {
	f := &File{Version: 1, Labs: map[string]*Attempt{
		"lab-a": {Passes: 2, LastPassHints: 0, LastPassed: true},
	}}
	f.StartAttempt("lab-a", 2, "", nil, nil)
	if !f.Get("lab-a").Mastered() {
		t.Error("starting a later attempt revoked established mastery")
	}
}

func TestRecordGradeTracksHintsFromLatestSuccessfulAttempt(t *testing.T) {
	f := &File{Version: 1, Labs: map[string]*Attempt{}}
	f.StartAttempt("lab-a", 1, "", nil, nil)
	f.RecordGrade("lab-a", true, true, time.Minute, 2, false)
	f.StartAttempt("lab-a", 2, "", nil, nil)
	f.RecordGrade("lab-a", true, true, time.Minute, 0, false)
	if !f.Get("lab-a").Mastered() {
		t.Error("a clean latest successful attempt did not establish mastery")
	}
	f.StartAttempt("lab-a", 3, "", nil, nil)
	f.RecordGrade("lab-a", false, false, time.Minute, 3, false)
	if !f.Get("lab-a").Mastered() {
		t.Error("a later failure rewrote the latest successful attempt's hints")
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
	f.MarkLesson("04-rbac", time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC))
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
	if !again.LessonViewed("04-rbac") {
		t.Error("lesson progress did not survive a round trip")
	}
}

func TestArchiveKeepsRecoverableHistoryAndStartsFresh(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DOJO_HOME", home)
	f := &File{Version: 1, Labs: map[string]*Attempt{}}
	f.StartAttempt("lab-a", 7, "v2", []string{"rbac"}, nil)
	if err := f.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	backup, err := Archive(time.Date(2026, 8, 25, 14, 3, 21, 0, time.UTC))
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	want := filepath.Join(home, "progress-20260825-140321.json")
	if backup != want {
		t.Errorf("backup = %q, want %q", backup, want)
	}
	if _, err := os.Stat(backup); err != nil {
		t.Fatalf("backup is not readable: %v", err)
	}
	fresh, err := Load()
	if err != nil {
		t.Fatalf("load fresh: %v", err)
	}
	if len(fresh.Labs) != 0 {
		t.Errorf("fresh history contains %d labs", len(fresh.Labs))
	}

	// A second reset in the same second must not overwrite the first backup.
	second, err := Archive(time.Date(2026, 8, 25, 14, 3, 21, 0, time.UTC))
	if err != nil {
		t.Fatalf("second archive: %v", err)
	}
	if second == backup {
		t.Error("second archive overwrote the first")
	}
}

// TestDevModeKeepsDogfoodingOutOfStudyHistory is the whole point of the split:
// proving an exercise works must not tell recommend, learn or readiness that
// the learner has practised it.
func TestDevModeKeepsDogfoodingOutOfStudyHistory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DOJO_HOME", home)

	t.Setenv("DOJO_DEV", "")
	real := &File{Version: 1, Labs: map[string]*Attempt{}}
	real.StartAttempt("rbac-follow", 1, "", []string{"rbac"}, nil)
	real.RecordGrade("rbac-follow", true, true, 27*time.Second, 0, false)
	if err := real.Save(); err != nil {
		t.Fatalf("save study history: %v", err)
	}

	t.Setenv("DOJO_DEV", "1")
	dev, err := Load()
	if err != nil {
		t.Fatalf("load dev history: %v", err)
	}
	if len(dev.Labs) != 0 {
		t.Fatalf("dev history read the study history: %d labs", len(dev.Labs))
	}
	dev.StartAttempt("etcd-snapshot", 2, "", []string{"etcd"}, nil)
	dev.RecordGrade("etcd-snapshot", true, true, 10*time.Second, 0, false)
	if err := dev.Save(); err != nil {
		t.Fatalf("save dev history: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "progress-dev.json")); err != nil {
		t.Fatalf("dev history was not written to progress-dev.json: %v", err)
	}

	// The dogfooded pass must be invisible to a study-mode read, and the
	// learner's own pass must have survived it.
	t.Setenv("DOJO_DEV", "")
	after, err := Load()
	if err != nil {
		t.Fatalf("reload study history: %v", err)
	}
	if _, ok := after.Labs["etcd-snapshot"]; ok {
		t.Error("a dogfooded attempt reached the study history")
	}
	if got := after.Get("rbac-follow").Passes; got != 1 {
		t.Errorf("study history lost its own pass: passes = %d", got)
	}
}

// TestDevArchiveDoesNotCollideWithStudyArchive keeps `dojo progress reset`
// honest in both modes: each history archives under its own name.
func TestDevArchiveDoesNotCollideWithStudyArchive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DOJO_HOME", home)
	t.Setenv("DOJO_DEV", "1")
	f := &File{Version: 1, Labs: map[string]*Attempt{}}
	f.StartAttempt("lab-a", 7, "", nil, nil)
	if err := f.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	backup, err := Archive(time.Date(2026, 9, 8, 7, 44, 56, 0, time.UTC))
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	want := filepath.Join(home, "progress-dev-20260908-074456.json")
	if backup != want {
		t.Errorf("backup = %q, want %q", backup, want)
	}
}
