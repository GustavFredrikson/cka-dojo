package curriculum

import (
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/content"
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
