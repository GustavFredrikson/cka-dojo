package lab

import (
	"strings"
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/content"
)

func repoContent(t *testing.T) *content.Source {
	t.Helper()
	src, err := content.Resolve("../..", "")
	if err != nil {
		t.Fatalf("resolve content: %v", err)
	}
	return src
}

const servicesLab = "curriculum/cka-2026/modules/05-services/labs/services-no-endpoints"

func TestLoadServicesLab(t *testing.T) {
	l, err := Load(repoContent(t), servicesLab)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if l.ID != "services-no-endpoints" {
		t.Errorf("id = %q", l.ID)
	}
	if l.Module != "05-services" {
		t.Errorf("module = %q, want 05-services", l.Module)
	}
	if l.Variants == nil || len(l.Variants.Options) != 3 {
		t.Fatalf("want 3 variants, got %+v", l.Variants)
	}
	task, err := l.Task()
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	// The task must describe the symptom, never the cause.
	for _, leak := range []string{"selector", "targetPort", "target port"} {
		if strings.Contains(strings.ToLower(task), strings.ToLower(leak)) {
			t.Errorf("task text gives away the fault by mentioning %q", leak)
		}
	}
}

func TestBuildEveryVariant(t *testing.T) {
	l, err := Load(repoContent(t), servicesLab)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, v := range l.Variants.Options {
		v := v
		t.Run(v.ID, func(t *testing.T) {
			plan, err := l.Build(&v)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			if len(plan.Faults) == 0 {
				t.Error("variant injects nothing")
			}
			if len(plan.Checks) == 0 {
				t.Error("variant grades nothing")
			}
			if len(plan.Manifests) == 0 {
				t.Error("variant applies no baseline")
			}
		})
	}
}

// TestPickVariantIsDeterministic is what makes `--seed` a reproduction tool
// rather than a suggestion.
func TestPickVariantIsDeterministic(t *testing.T) {
	l, err := Load(repoContent(t), servicesLab)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	first := l.PickVariant(9182731)
	for i := 0; i < 20; i++ {
		if got := l.PickVariant(9182731); got.ID != first.ID {
			t.Fatalf("same seed gave %q then %q", first.ID, got.ID)
		}
	}
	// And different seeds must actually reach different variants.
	seen := map[string]bool{}
	for seed := int64(1); seed < 200; seed++ {
		seen[l.PickVariant(seed).ID] = true
	}
	if len(seen) != len(l.Variants.Options) {
		t.Errorf("seeds reached %d of %d variants", len(seen), len(l.Variants.Options))
	}
}

func TestValidateCatchesUnknownTypes(t *testing.T) {
	l, err := Load(repoContent(t), servicesLab)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	errs := l.Validate(
		map[string]bool{"services-networking": true, "troubleshooting": true},
		map[string]bool{"services": true}, // deliberately incomplete
		map[string]bool{"standard": true},
	)
	if len(errs) == 0 {
		t.Fatal("undeclared skills were not reported")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "unknown skill") {
			found = true
		}
	}
	if !found {
		t.Errorf("errors did not mention the unknown skill: %v", errs)
	}
}

func TestNamespaceNames(t *testing.T) {
	manifest := `apiVersion: v1
kind: Namespace
metadata:
  name: shop
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: shop
`
	got := namespaceNames(manifest)
	if len(got) != 1 || got[0] != "shop" {
		t.Errorf("namespaceNames = %v, want [shop]", got)
	}
}
