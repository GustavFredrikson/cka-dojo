package readiness

import (
	"strings"
	"testing"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/curriculum"
	"github.com/gustavfredrikson/cka-dojo/internal/lab"
	"github.com/gustavfredrikson/cka-dojo/internal/progress"
)

var now = time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)

// twoDomains is the smallest curriculum that can show the untouched-domain
// gate: one heavy domain and one light one, two labs each.
func twoDomains() *curriculum.Curriculum {
	c := &curriculum.Curriculum{
		Domains: []curriculum.Domain{
			{ID: "troubleshooting", Name: "Troubleshooting", Weight: 90},
			{ID: "storage", Name: "Storage", Weight: 10},
		},
	}
	c.Modules = []*curriculum.Module{{
		ID: "m",
		Labs: []*lab.Lab{
			{ID: "ts-1", Domains: []string{"troubleshooting"}, TargetMinutes: 5},
			{ID: "ts-2", Domains: []string{"troubleshooting"}, TargetMinutes: 5},
			{ID: "st-1", Domains: []string{"storage"}, TargetMinutes: 5},
			{ID: "st-2", Domains: []string{"storage"}, TargetMinutes: 5},
		},
	}}
	return c
}

func mastered(at time.Time) *progress.Attempt {
	return &progress.Attempt{Attempts: 2, Passes: 2, CleanPasses: 2, LastPassed: true, LastAt: at}
}

func history(labs map[string]*progress.Attempt) *progress.File {
	return &progress.File{Version: 1, Labs: labs}
}

// An untouched domain is the whole point of the command: 90% of the exam
// mastered must not read as ready while the other 10% has never been opened.
func TestUntouchedDomainBlocksReadyRegardlessOfScore(t *testing.T) {
	r := Assess(twoDomains(), history(map[string]*progress.Attempt{
		"ts-1": mastered(now), "ts-2": mastered(now),
	}), now)

	if r.Score < 0.89 {
		t.Fatalf("score = %.2f, want the heavy domain to carry it", r.Score)
	}
	if r.Verdict != VerdictNotReady {
		t.Errorf("verdict = %q, want %q", r.Verdict, VerdictNotReady)
	}

	steps := r.NextSteps()
	if len(steps) == 0 || !strings.Contains(steps[0], "Storage") {
		t.Errorf("first step = %q, want the untouched domain first", steps)
	}
}

func TestEverythingMasteredIsLikelyReady(t *testing.T) {
	r := Assess(twoDomains(), history(map[string]*progress.Attempt{
		"ts-1": mastered(now), "ts-2": mastered(now),
		"st-1": mastered(now), "st-2": mastered(now),
	}), now)

	if r.Verdict != VerdictLikelyReady {
		t.Errorf("verdict = %q, want %q", r.Verdict, VerdictLikelyReady)
	}
	if r.Score != 1 {
		t.Errorf("score = %.2f, want 1", r.Score)
	}
	if len(r.NextSteps()) != 0 {
		t.Errorf("next steps = %v, want none", r.NextSteps())
	}
}

// A single pass is exposure, not competence, and must not score.
func TestSinglePassDoesNotCountAsMastery(t *testing.T) {
	r := Assess(twoDomains(), history(map[string]*progress.Attempt{
		"ts-1": {Attempts: 1, Passes: 1, CleanPasses: 1, LastPassed: true, LastAt: now},
		"ts-2": {Attempts: 1, Passes: 1, CleanPasses: 1, LastPassed: true, LastAt: now},
		"st-1": {Attempts: 1, Passes: 1, CleanPasses: 1, LastPassed: true, LastAt: now},
		"st-2": {Attempts: 1, Passes: 1, CleanPasses: 1, LastPassed: true, LastAt: now},
	}), now)

	if r.Score != 0 {
		t.Errorf("score = %.2f, want 0", r.Score)
	}
	if len(r.Unproven) != 4 {
		t.Errorf("unproven = %d, want 4", len(r.Unproven))
	}
	if r.Verdict != VerdictNotReady {
		t.Errorf("verdict = %q, want %q", r.Verdict, VerdictNotReady)
	}
}

// Each lab lands in exactly one bucket, so the counts in "do this next" add up
// rather than double-reporting the same lab as both unproven and hinted.
func TestLabCategoriesAreMutuallyExclusive(t *testing.T) {
	r := Assess(twoDomains(), history(map[string]*progress.Attempt{
		"ts-1": {Attempts: 3, Passes: 0, LastAt: now},
		"ts-2": {Attempts: 2, Passes: 1, CleanPasses: 0, LastPassed: true, LastAt: now},
		"st-1": {Attempts: 1, Passes: 1, CleanPasses: 1, LastPassed: true, LastAt: now},
		"st-2": mastered(now),
	}), now)

	if len(r.Failing) != 1 || r.Failing[0] != "ts-1" {
		t.Errorf("failing = %v, want [ts-1]", r.Failing)
	}
	if len(r.HintDependent) != 1 || r.HintDependent[0] != "ts-2" {
		t.Errorf("hintDependent = %v, want [ts-2]", r.HintDependent)
	}
	if len(r.Unproven) != 1 || r.Unproven[0] != "st-1" {
		t.Errorf("unproven = %v, want [st-1]", r.Unproven)
	}
}

func TestStaleMasteryCountsHalf(t *testing.T) {
	old := now.Add(-staleAfter - time.Hour)
	r := Assess(twoDomains(), history(map[string]*progress.Attempt{
		"ts-1": mastered(old), "ts-2": mastered(old),
		"st-1": mastered(now), "st-2": mastered(now),
	}), now)

	ts := find(t, r, "troubleshooting")
	if ts.Stale != 2 {
		t.Errorf("stale = %d, want 2", ts.Stale)
	}
	if ts.Coverage != 0.5 {
		t.Errorf("coverage = %.2f, want 0.50", ts.Coverage)
	}
	// 0.5*90 + 1.0*10 over 100.
	if r.Score != 0.55 {
		t.Errorf("score = %.2f, want 0.55", r.Score)
	}
}

func TestSlowRunsAreReportedAgainstTarget(t *testing.T) {
	r := Assess(twoDomains(), history(map[string]*progress.Attempt{
		// 5m target: 451s is over 1.5x, 400s is not.
		"ts-1": {Attempts: 1, Passes: 1, CleanPasses: 1, LastSeconds: 451, LastAt: now},
		"ts-2": {Attempts: 1, Passes: 1, CleanPasses: 1, LastSeconds: 400, LastAt: now},
	}), now)

	if len(r.Slow) != 1 || r.Slow[0].Lab != "ts-1" {
		t.Fatalf("slow = %+v, want only ts-1", r.Slow)
	}
	if r.Slow[0].Target != 300 {
		t.Errorf("target = %d, want 300", r.Slow[0].Target)
	}
}

// A domain the curriculum declares but has written no labs for must not be
// scored as complete, or an unwritten module would raise the verdict.
func TestDomainWithNoLabsIsExcludedFromScore(t *testing.T) {
	c := twoDomains()
	c.Domains = append(c.Domains, curriculum.Domain{ID: "unwritten", Name: "Unwritten", Weight: 0})
	r := Assess(c, history(map[string]*progress.Attempt{
		"ts-1": mastered(now), "ts-2": mastered(now),
		"st-1": mastered(now), "st-2": mastered(now),
	}), now)

	if r.Verdict != VerdictLikelyReady {
		t.Errorf("verdict = %q, want %q", r.Verdict, VerdictLikelyReady)
	}
	d := find(t, r, "unwritten")
	if d.Coverage != 0 || d.Labs != 0 {
		t.Errorf("unwritten domain = %+v, want zero labs and coverage", d)
	}
}

func TestKnownGapsAreCarriedIntoTheReport(t *testing.T) {
	c := twoDomains()
	c.KnownGaps = []curriculum.Gap{{ID: "etcd", Name: "etcd", Domain: "troubleshooting", Note: "none"}}
	r := Assess(c, history(nil), now)
	if len(r.Uncovered) != 1 || r.Uncovered[0].ID != "etcd" {
		t.Errorf("uncovered = %+v, want the curriculum's gap", r.Uncovered)
	}
}

func find(t *testing.T, r *Report, id string) Domain {
	t.Helper()
	for _, d := range r.Domains {
		if d.ID == id {
			return d
		}
	}
	t.Fatalf("domain %q missing from report", id)
	return Domain{}
}
