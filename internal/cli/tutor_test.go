package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/config"
	"github.com/gustavfredrikson/cka-dojo/internal/curriculum"
	"github.com/gustavfredrikson/cka-dojo/internal/lab"
)

func TestTutorContextAllowListDoesNotLeakScenarioInternals(t *testing.T) {
	cur := &curriculum.Curriculum{ID: "cka-2026"}
	cur.Kubernetes.Minor = "1.35"
	cur.Skills = []curriculum.Skill{{ID: "services", Name: "Services"}}
	module := &curriculum.Module{ID: "05-services", Name: "Services"}
	exercise := &lab.Lab{
		ID: "services-no-endpoints", Title: "A Service that does not answer",
		Skills: []string{"services"}, TargetMinutes: 6,
	}
	state := &config.State{
		Profile: "standard", Mode: config.ModePractice,
		Variant: "SECRET-selector-mismatch", Seed: 9182731,
		StartedAt: time.Now().Add(-3 * time.Minute), HintsUsed: 1, Attempt: 2,
	}

	got := formatTutorContext(cur, module, exercise, state)
	for _, want := range []string{
		"Curriculum: cka-2026", "Kubernetes: 1.35",
		"Active lab: services-no-endpoints", "Skills: Services",
		"Hints used: 1", "Attempt: 2",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("context is missing %q:\n%s", want, got)
		}
	}
	for _, leak := range []string{"SECRET-selector-mismatch", "9182731", "grader", "faults:"} {
		if strings.Contains(got, leak) {
			t.Errorf("context leaked %q:\n%s", leak, got)
		}
	}
}
