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

func TestNodeIPArg(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		// The shape the provisioning script actually writes: one assignment,
		// with the flag embedded in it rather than standing alone.
		{"as provisioned", "KUBELET_EXTRA_ARGS=--node-ip=192.168.104.5\n", "192.168.104.5"},
		{"quoted value", `KUBELET_EXTRA_ARGS="--node-ip=10.0.0.1"`, "10.0.0.1"},
		{"further args after it", `KUBELET_EXTRA_ARGS="--node-ip=10.0.0.1 --v=2"`, "10.0.0.1"},
		{"flag stands alone", "--node-ip=10.0.0.2", "10.0.0.2"},
		{"other args only", "KUBELET_EXTRA_ARGS=--v=4\n", ""},
		{"file absent", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := nodeIPArg(c.in); got != c.want {
				t.Errorf("nodeIPArg(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestUpRefusesWhenANodeAddressMoved covers the guard added after a cluster was
// found unrecoverable because cp1's address had changed underneath it. Without
// the guard the symptom was a ten-minute wait in WaitReady and a timeout that
// named nothing; the point of the check is that it fails at once and says what
// to do.
func TestUpRefusesWhenANodeAddressMoved(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())

	prof := minimalProfile()
	prov := fake.New()
	// cp1 reports one address and insists, through its own kubelet defaults,
	// that it was provisioned with another.
	prov.Responder = func(node, script string) (provider.ExecResult, error) {
		switch {
		case strings.Contains(script, "ip -4 -o addr show"):
			return provider.ExecResult{Stdout: "192.168.104.4\n"}, nil
		case strings.Contains(script, "/etc/default/kubelet"):
			return provider.ExecResult{Stdout: "KUBELET_EXTRA_ARGS=--node-ip=192.168.104.5\n"}, nil
		case strings.Contains(script, "test -f"):
			return provider.ExecResult{Stdout: "no\n"}, nil
		}
		return provider.ExecResult{}, nil
	}
	m := New(prof, prov, repoContent(t))

	err := m.Up(context.Background(), false)
	if err == nil {
		t.Fatal("Up succeeded on an environment whose node addresses had moved")
	}
	for _, want := range []string{"cp1", "192.168.104.5", "192.168.104.4", "dojo env reset"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %q:\n%v", want, err)
		}
	}
	// It must refuse before touching the cluster, not after half-provisioning.
	if prov.Ran("kubeadm init") {
		t.Error("Up ran kubeadm init despite the address mismatch")
	}
}

// TestVerifyNodeAddressesAcceptsAHealthyEnvironment is the other half of the
// guard: it must be silent when the addresses agree, and on a freshly created
// environment where no node has been provisioned yet and there is no recorded
// address to compare against. Called directly rather than through Up, which
// would drag the whole kubeadm bring-up in and test the fake, not the guard.
func TestVerifyNodeAddressesAcceptsAHealthyEnvironment(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())

	for _, tc := range []struct {
		name, kubeletDefaults string
	}{
		{"addresses agree", "KUBELET_EXTRA_ARGS=--node-ip=192.168.104.4\n"},
		{"never provisioned", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prov := fake.New()
			prov.Responder = func(node, script string) (provider.ExecResult, error) {
				switch {
				case strings.Contains(script, "ip -4 -o addr show"):
					return provider.ExecResult{Stdout: "192.168.104.4\n"}, nil
				case strings.Contains(script, "/etc/default/kubelet"):
					return provider.ExecResult{Stdout: tc.kubeletDefaults}, nil
				}
				return provider.ExecResult{}, nil
			}
			m := New(minimalProfile(), prov, repoContent(t))
			if err := m.verifyNodeAddresses(context.Background()); err != nil {
				t.Errorf("verifyNodeAddresses: %v", err)
			}
		})
	}
}
