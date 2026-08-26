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

func TestSchedulingModuleIsAnOrderedFiveStagePath(t *testing.T) {
	src, err := content.Resolve("../..", "")
	if err != nil {
		t.Fatalf("content: %v", err)
	}
	cur, err := Load(src, "cka-2026")
	if err != nil {
		t.Fatalf("curriculum: %v", err)
	}
	module := cur.ModuleByID("scheduling")
	if module == nil || len(module.Labs) != 5 {
		t.Fatalf("scheduling module = %#v, want five exercises", module)
	}
	for i, exercise := range module.Labs {
		info, ok := exercise.LearningStage.Info()
		if !ok || info.Level != i+1 {
			t.Errorf("exercise %s stage = %+v, %v; want level %d", exercise.ID, info, ok, i+1)
		}
		if i > 0 && (len(exercise.Prerequisites) != 1 || exercise.Prerequisites[0] != module.Labs[i-1].ID) {
			t.Errorf("exercise %s does not require previous stage %s", exercise.ID, module.Labs[i-1].ID)
		}
	}
	contextual := module.Labs[4]
	if contextual.Variants == nil || len(contextual.Variants.Options) != 2 {
		t.Fatalf("contextual variants = %#v, want selector and taint", contextual.Variants)
	}
	for i := range contextual.Variants.Options {
		if _, err := contextual.Build(&contextual.Variants.Options[i]); err != nil {
			t.Errorf("build variant %s: %v", contextual.Variants.Options[i].ID, err)
		}
	}
}

func TestExpandedCurriculumHasFortyEightBuildableExercises(t *testing.T) {
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
		"scheduling":           5,
		"rbac":                 5,
		"services":             7,
		"dns-networking":       3,
		"network-policy":       5,
		"storage":              5,
		"control-plane":        3,
		"packaging-extensions": 3,
		"troubleshooting":      2,
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
	if total != 48 {
		t.Errorf("curriculum has %d exercises, want 48", total)
	}
}
