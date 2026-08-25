package environment

import (
	"context"
	"strings"
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/provider"
	"github.com/gustavfredrikson/cka-dojo/internal/provider/fake"
)

// TestBaseProvisionWiresNodesTogether covers the parts of bring-up that have
// nothing to do with Kubernetes but everything to do with the labs working:
// name resolution between machines and SSH from the workstation.
func TestBaseProvisionWiresNodesTogether(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())

	prof := minimalProfile()
	prof.Provisioning = ProvisionNone
	prov := fake.New()
	m := New(prof, prov, repoContent(t))

	if err := m.Up(context.Background(), false); err != nil {
		t.Fatalf("Up: %v", err)
	}

	for _, node := range []string{"terminal", "cp1"} {
		if st, _ := prov.Status(context.Background(), m.VMName(node)); st != "running" {
			t.Errorf("%s status = %q, want running", node, st)
		}
	}

	// Every node must be able to resolve every other node by short name.
	if !prov.Ran("# BEGIN dojo") {
		t.Error("no /etc/hosts block was written")
	}
	if !prov.Ran("192.168.104.") {
		t.Error("/etc/hosts block has no discovered addresses")
	}

	// The workstation gets a private key; every node authorises the public one.
	keyPath := "/home/" + StudentUser + "/.ssh/id_ed25519"
	if _, ok := prov.Files[m.VMName("terminal")+":"+keyPath]; !ok {
		t.Error("no SSH key was installed on the workstation")
	}
	if _, ok := prov.Files[m.VMName("cp1")+":"+keyPath]; ok {
		t.Error("the private key leaked onto a cluster node")
	}
	if !prov.Ran("authorized_keys") {
		t.Error("the workstation key was not authorised on the nodes")
	}
	cfg, ok := prov.Files[m.VMName("terminal")+":/home/"+StudentUser+"/.ssh/config"]
	if !ok {
		t.Fatal("no ssh config on the workstation")
	}
	if !strings.Contains(string(cfg), "Host cp1") {
		t.Errorf("ssh config does not reach cp1:\n%s", cfg)
	}
}

// TestStepsAreSkippedWhenAlreadyDone proves setup is resumable: a completed
// step leaves a marker, and a second run does not redo it.
func TestStepsAreSkippedWhenAlreadyDone(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())

	prof := minimalProfile()
	prof.Provisioning = ProvisionNone
	prov := fake.New()
	m := New(prof, prov, repoContent(t))

	if err := m.Up(context.Background(), false); err != nil {
		t.Fatalf("first Up: %v", err)
	}
	if countScripts(prov, "apt-get") == 0 {
		t.Fatal("base packages were never installed")
	}

	// Second run: every marker now reports as present.
	before := countScripts(prov, "apt-get")
	prov.Responder = func(node, script string) (provider.ExecResult, error) {
		switch {
		case strings.Contains(script, "ip -4 -o addr show"):
			return provider.ExecResult{Stdout: "192.168.104.11\n"}, nil
		case strings.Contains(script, "test -f"):
			return provider.ExecResult{Stdout: "yes\n"}, nil
		default:
			return provider.ExecResult{}, nil
		}
	}
	if err := m.Up(context.Background(), false); err != nil {
		t.Fatalf("second Up: %v", err)
	}
	if after := countScripts(prov, "apt-get"); after != before {
		t.Errorf("apt-get ran again on a resumed setup: %d -> %d", before, after)
	}

	// ...unless the caller asks for it.
	if err := m.Up(context.Background(), true); err != nil {
		t.Fatalf("forced Up: %v", err)
	}
	if after := countScripts(prov, "apt-get"); after == before {
		t.Error("--force did not re-run completed steps")
	}
}

func countScripts(prov *fake.Provider, substr string) int {
	n := 0
	for _, c := range prov.Calls {
		if strings.Contains(c.Script, substr) {
			n++
		}
	}
	return n
}
