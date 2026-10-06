package curriculum

import (
	"strings"
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/content"
	"github.com/gustavfredrikson/cka-dojo/internal/lab"
)

func TestEveryCurrentModuleHasAStructuredLesson(t *testing.T) {
	src, err := content.Resolve("../..", "")
	if err != nil {
		t.Fatalf("content: %v", err)
	}
	cur, err := Load(src, "cka-2026")
	if err != nil {
		t.Fatalf("curriculum: %v", err)
	}
	for _, module := range cur.Modules {
		if errs := module.ValidateLesson(src); len(errs) > 0 {
			t.Errorf("%s lesson: %v", module.ID, errs)
		}
	}
}

func TestValidateRejectsUnknownPrerequisite(t *testing.T) {
	c := &Curriculum{
		SchemaVersion: 1,
		Domains:       []Domain{{ID: "services-networking", Weight: 100}},
		Modules: []*Module{{ID: "services", Labs: []*lab.Lab{{
			ID: "later", Prerequisites: []string{"missing"},
		}}}},
	}
	c.Kubernetes.Minor = "1.35"
	errs := c.Validate()
	for _, err := range errs {
		if strings.Contains(err.Error(), "unknown prerequisite") {
			return
		}
	}
	t.Fatalf("errors did not reject unknown prerequisite: %v", errs)
}

func TestValidateRejectsPrerequisiteCycle(t *testing.T) {
	c := &Curriculum{
		SchemaVersion: 1,
		Domains:       []Domain{{ID: "services-networking", Weight: 100}},
		Modules: []*Module{{ID: "services", Labs: []*lab.Lab{
			{ID: "a", Prerequisites: []string{"b"}},
			{ID: "b", Prerequisites: []string{"a"}},
		}}},
	}
	c.Kubernetes.Minor = "1.35"
	for _, err := range c.Validate() {
		if strings.Contains(err.Error(), "prerequisite cycle") {
			return
		}
	}
	t.Fatal("prerequisite cycle passed validation")
}

func TestModuleLookupUsesFriendlyTopicNames(t *testing.T) {
	c := &Curriculum{Modules: []*Module{
		{ID: "04-rbac", Name: "RBAC"},
		{ID: "05-services", Name: "Services"},
	}}
	for query, want := range map[string]string{
		"services":    "05-services",
		"05-services": "05-services",
		"rbac":        "04-rbac",
	} {
		got := c.ModuleByID(query)
		if got == nil || got.ID != want {
			t.Errorf("ModuleByID(%q) = %#v, want %s", query, got, want)
		}
	}
}

// TestSchedulingModuleClimbsItsStagesInOrder pins the property that makes a
// module teachable: reading it top to bottom never asks for a stage of
// independence the learner has not reached yet, and never names a
// prerequisite that comes later in the same module. A module may carry more
// than one exercise at a stage — scheduling covers both affinity and taints
// at build and contextual-fix — so the stages must not decrease rather than
// having to increase by exactly one.
func TestSchedulingModuleClimbsItsStagesInOrder(t *testing.T) {
	src, err := content.Resolve("../..", "")
	if err != nil {
		t.Fatalf("content: %v", err)
	}
	cur, err := Load(src, "cka-2026")
	if err != nil {
		t.Fatalf("curriculum: %v", err)
	}
	module := cur.ModuleByID("scheduling")
	if module == nil {
		t.Fatal("no scheduling module")
	}

	seen := map[string]bool{}
	previous := 0
	stages := map[int]bool{}
	for i, exercise := range module.Labs {
		info, ok := exercise.LearningStage.Info()
		if !ok {
			t.Errorf("exercise %s has unknown stage %q", exercise.ID, exercise.LearningStage)
			continue
		}
		if info.Level < previous {
			t.Errorf("exercise %s (stage %d) comes after a stage %d exercise", exercise.ID, info.Level, previous)
		}
		previous = info.Level
		stages[info.Level] = true
		for _, req := range exercise.Prerequisites {
			if !seen[req] {
				t.Errorf("exercise %s (position %d) requires %s, which is not earlier in the module", exercise.ID, i+1, req)
			}
		}
		seen[exercise.ID] = true
	}

	// Follow through contextual-fix must all still be represented: that ladder
	// is the point of the module.
	for level := 1; level <= 5; level++ {
		if !stages[level] {
			t.Errorf("scheduling module has no stage %d exercise", level)
		}
	}

	contextual, _ := cur.LabByID("scheduling-pending")
	if contextual == nil {
		t.Fatal("no scheduling-pending exercise")
	}
	if contextual.Variants == nil || len(contextual.Variants.Options) != 2 {
		t.Fatalf("contextual variants = %#v, want selector and taint", contextual.Variants)
	}
	for i := range contextual.Variants.Options {
		if _, err := contextual.Build(&contextual.Variants.Options[i]); err != nil {
			t.Errorf("build variant %s: %v", contextual.Variants.Options[i].ID, err)
		}
	}
}

// TestCurriculumHasTheExpectedBuildableExercises is a deliberate snapshot: it
// has to be edited in the same commit as any new exercise, which is what stops
// content from being added without also being counted and dogfooded.
func TestCurriculumHasTheExpectedBuildableExercises(t *testing.T) {
	src, err := content.Resolve("../..", "")
	if err != nil {
		t.Fatalf("content: %v", err)
	}
	cur, err := Load(src, "cka-2026")
	if err != nil {
		t.Fatalf("curriculum: %v", err)
	}

	wantByModule := map[string]int{
		"workloads":            10,
		"scheduling":           7,
		"rbac":                 7,
		"services":             8,
		"dns-networking":       5,
		"network-policy":       5,
		"ingress":              5,
		"storage":              8,
		"admission":            4,
		"workload-primitives":  6,
		"control-plane":        12,
		"packaging-extensions": 4,
		"observability":        2,
		"troubleshooting":      3,
		"cluster-bootstrap":    4,
		"cluster-upgrade":      3,
		"ha-control-plane":     3,
	}
	total := 0
	for topic, want := range wantByModule {
		module := cur.ModuleByID(topic)
		if module == nil {
			t.Errorf("missing module %q", topic)
			continue
		}
		if got := len(module.Labs); got != want {
			t.Errorf("%s has %d exercises, want %d", topic, got, want)
		}
		total += len(module.Labs)
		for _, exercise := range module.Labs {
			if exercise.Variants == nil {
				continue
			}
			for i := range exercise.Variants.Options {
				variant := &exercise.Variants.Options[i]
				if _, err := exercise.Build(variant); err != nil {
					t.Errorf("build %s variant %s: %v", exercise.ID, variant.ID, err)
				}
			}
		}
	}
	if want := 96; total != want {
		t.Errorf("curriculum has %d exercises, want %d", total, want)
	}
	if got := cur.LabCount(); got != total {
		t.Errorf("%d exercises are reachable through modules, but the curriculum holds %d", total, got)
	}
}
