package recommend

import (
	"testing"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/curriculum"
	"github.com/gustavfredrikson/cka-dojo/internal/lab"
	"github.com/gustavfredrikson/cka-dojo/internal/progress"
)

func testCurriculum() *curriculum.Curriculum {
	c := &curriculum.Curriculum{
		Domains: []curriculum.Domain{
			{ID: "troubleshooting", Weight: 30},
			{ID: "storage", Weight: 10},
		},
	}
	c.Modules = []*curriculum.Module{{
		ID: "mixed",
		Labs: []*lab.Lab{
			{ID: "failed-high-weight", Domains: []string{"troubleshooting"}},
			{ID: "new-storage", Domains: []string{"storage"}},
			{ID: "mastered-high-weight", Domains: []string{"troubleshooting"}},
		},
	}}
	return c
}

func TestRankUsesWeightMasteryAndRecency(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	history := &progress.File{Version: 1, Labs: map[string]*progress.Attempt{
		"failed-high-weight": {
			Attempts: 2, Passes: 0, LastAt: now.Add(-7 * 24 * time.Hour),
		},
		"mastered-high-weight": {
			Attempts: 3, Passes: 2, LastPassed: true, LastHints: 0,
			LastAt: now.Add(-24 * time.Hour),
		},
	}}

	got := Rank(testCurriculum(), history, now)
	if len(got) != 3 {
		t.Fatalf("recommendations = %d, want 3", len(got))
	}
	if got[0].Lab.ID != "failed-high-weight" {
		t.Errorf("first = %s, want failed-high-weight", got[0].Lab.ID)
	}
	if got[0].Reason != "attempted but not passed" {
		t.Errorf("reason = %q", got[0].Reason)
	}
	if got[2].Lab.ID != "mastered-high-weight" {
		t.Errorf("last = %s, want mastered-high-weight", got[2].Lab.ID)
	}
}

func TestAverageWeightDoesNotDoubleCountMultiDomainLabs(t *testing.T) {
	got := averageWeight([]string{"a", "b"}, map[string]int{"a": 30, "b": 20})
	if got != 25 {
		t.Errorf("averageWeight = %.1f, want 25", got)
	}
}

func TestRecencyRisesAndCaps(t *testing.T) {
	now := time.Now()
	a := &progress.Attempt{LastAt: now}
	if got := recencyFor(a, now); got != 0.75 {
		t.Errorf("immediate recency = %.2f, want .75", got)
	}
	old := &progress.Attempt{LastAt: now.Add(-90 * 24 * time.Hour)}
	if got := recencyFor(old, now); got != 1.25 {
		t.Errorf("old recency = %.2f, want 1.25", got)
	}
}

func TestLockedExerciseIsNotRecommended(t *testing.T) {
	now := time.Now()
	c := testCurriculum()
	locked := c.Modules[0].Labs[0]
	locked.Prerequisites = []string{"foundation"}
	history := &progress.File{Version: 1, Labs: map[string]*progress.Attempt{}}

	got := Rank(c, history, now)
	for _, recommendation := range got {
		if recommendation.Lab.ID != locked.ID {
			continue
		}
		if recommendation.Unlocked || recommendation.Score != 0 {
			t.Fatalf("locked recommendation = unlocked %v, score %.1f", recommendation.Unlocked, recommendation.Score)
		}
		return
	}
	t.Fatal("locked exercise disappeared from full ranking")
}

func TestEarlyStageGetsProgressionPreference(t *testing.T) {
	if stageFactor(lab.StageFollow) <= stageFactor(lab.StageDiagnose) {
		t.Fatal("follow stage should rank ahead of blind diagnosis when other factors match")
	}
}
