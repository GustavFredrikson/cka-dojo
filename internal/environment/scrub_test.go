package environment

import (
	"context"
	"strings"
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/provider/fake"
)

// TestScrubWipesResidueButNotPlumbing is the safety net for a destructive
// command: it must clear the learner's own files while leaving behind the
// three things whose loss would break the environment or `dojo reset`.
func TestScrubWipesResidueButNotPlumbing(t *testing.T) {
	prov := fake.New()
	m := New(minimalProfile(), prov, repoContent(t))

	if err := m.Scrub(context.Background(), []string{"terminal", "cp1"}, ScrubOptions{}); err != nil {
		t.Fatalf("Scrub: %v", err)
	}

	if len(prov.Calls) != 2 {
		t.Fatalf("scrub made %d calls, want one per node", len(prov.Calls))
	}
	for _, c := range prov.Calls {
		if c.User != StudentUser {
			t.Errorf("scrub on %s ran as %q, want %q", c.Node, c.User, StudentUser)
		}
		for _, keep := range []string{".ssh", ".kube"} {
			if !strings.Contains(c.Script, keep) {
				t.Errorf("scrub on %s does not spare %s", c.Node, keep)
			}
		}
		if !strings.Contains(c.Script, "rm -rf") {
			t.Errorf("scrub on %s removes nothing", c.Node)
		}
		if !strings.Contains(c.Script, "/etc/skel") {
			t.Errorf("scrub on %s leaves a stripped account, not a fresh one", c.Node)
		}
		// dojo keeps fault backups and step markers under /var/lib/dojo.
		// Scrubbing those would silently break lab resets.
		if strings.Contains(c.Script, "/var/lib/dojo") {
			t.Errorf("scrub on %s reaches into dojo's own state", c.Node)
		}
		if strings.Contains(c.Script, ShellDefaultsMarker) {
			t.Errorf("scrub on %s re-seeded the aliases without --defaults", c.Node)
		}
	}
}

// TestScrubDryRunRemovesNothing keeps --dry-run honest.
func TestScrubDryRunRemovesNothing(t *testing.T) {
	prov := fake.New()
	m := New(minimalProfile(), prov, repoContent(t))

	if err := m.Scrub(context.Background(), []string{"terminal"}, ScrubOptions{DryRun: true, Defaults: true}); err != nil {
		t.Fatalf("Scrub: %v", err)
	}
	script := prov.Calls[0].Script
	for _, mutation := range []string{"rm -rf", "install -m", ">>"} {
		if strings.Contains(script, mutation) {
			t.Errorf("dry run would still run %q:\n%s", mutation, script)
		}
	}
	if !strings.Contains(script, "remove ~/") {
		t.Error("dry run reports nothing")
	}
}

// TestScrubDefaultsMatchProvisioning is the anti-drift test. An improved
// alias block only reaches an already-provisioned machine through a scrub,
// because terminal.sh guards on the marker and never rewrites the block. If
// these two texts diverge, a re-seed pins the learner to the old version.
func TestScrubDefaultsMatchProvisioning(t *testing.T) {
	prov := fake.New()
	m := New(minimalProfile(), prov, repoContent(t))

	defaults, err := m.ShellDefaults()
	if err != nil {
		t.Fatalf("ShellDefaults: %v", err)
	}
	if !strings.Contains(defaults, ShellDefaultsMarker) {
		t.Errorf("the defaults block lacks the marker terminal.sh greps for:\n%s", defaults)
	}

	v, err := m.vars(context.Background(), "terminal")
	if err != nil {
		t.Fatalf("vars: %v", err)
	}
	body, err := m.renderScript("terminal.sh", v)
	if err != nil {
		t.Fatalf("render terminal.sh: %v", err)
	}
	if !strings.Contains(body, defaults) {
		t.Errorf("terminal.sh installs a different block than a scrub re-seeds:\n--- provisioning ---\n%s\n--- scrub ---\n%s", body, defaults)
	}

	if err := m.Scrub(context.Background(), []string{"terminal"}, ScrubOptions{Defaults: true}); err != nil {
		t.Fatalf("Scrub: %v", err)
	}
	// vars() above discovered addresses through the same fake, so the
	// scrub is the last call rather than the first.
	last := prov.Calls[len(prov.Calls)-1].Script
	if !strings.Contains(last, defaults) {
		t.Errorf("--defaults did not re-seed the block:\n%s", last)
	}
}

// TestScrubRejectsUnknownNode fails before touching any node, so a typo in
// --node cannot scrub the wrong machine.
func TestScrubRejectsUnknownNode(t *testing.T) {
	prov := fake.New()
	m := New(minimalProfile(), prov, repoContent(t))

	err := m.Scrub(context.Background(), []string{"terminal", "worker9"}, ScrubOptions{})
	if err == nil {
		t.Fatal("scrub accepted a node that does not exist")
	}
	if len(prov.Calls) != 0 {
		t.Errorf("scrub ran on %d node(s) before rejecting the unknown one", len(prov.Calls))
	}
}
