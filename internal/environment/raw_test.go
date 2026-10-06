package environment

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/provider"
	"github.com/gustavfredrikson/cka-dojo/internal/provider/fake"
)

// TestRawStopsBeforeKubeadm is the definition of the `nodes` provisioning
// mode: the machines arrive prepared and nothing is initialised, because
// initialising it is the exercise.
func TestRawStopsBeforeKubeadm(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())

	prof, err := LoadProfile(repoContent(t), "raw")
	if err != nil {
		t.Fatalf("load raw profile: %v", err)
	}
	prov := fake.New()
	m := New(prof, prov, repoContent(t))

	if err := m.Up(context.Background(), false); err != nil {
		t.Fatalf("Up: %v", err)
	}

	for _, forbidden := range []string{"kubeadm init", "kubeadm join", "calico", "metrics-server"} {
		if prov.Ran(forbidden) {
			t.Errorf("raw provisioning ran %q; it must stop before the cluster exists", forbidden)
		}
	}
	// But the toolchain must be there, or the learner is installing containerd
	// rather than practising kubeadm.
	if !prov.Ran("apt-mark hold kubelet kubeadm kubectl") {
		t.Error("raw provisioning did not install and pin the kubeadm toolchain")
	}
}

// TestRawPreparesClusterNodesOnly guards the role split: the workstation runs
// no kubelet, which is what lets a lab break the control plane without
// breaking the shell the learner is working from.
func TestRawPreparesClusterNodesOnly(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())

	prof, err := LoadProfile(repoContent(t), "raw")
	if err != nil {
		t.Fatalf("load raw profile: %v", err)
	}
	prov := fake.New()
	m := New(prof, prov, repoContent(t))
	if err := m.Up(context.Background(), false); err != nil {
		t.Fatalf("Up: %v", err)
	}

	for _, node := range []string{"cp1", "worker1"} {
		if !ranOn(prov, m.VMName(node), "apt-mark hold kubelet") {
			t.Errorf("%s did not get the kubeadm toolchain", node)
		}
	}
	if ranOn(prov, m.VMName("terminal"), "apt-mark hold kubelet") {
		t.Error("the workstation was prepared as a cluster node; it runs no kubelet")
	}
}

// TestRawKeepsAControlPlaneForGraders is the one-line regression guard on the
// decision the whole profile turns on. With no control-plane node,
// Profile.ControlPlane() is nil and every kubectl grader reports a broken
// engine rather than a failed check -- forever, including after the learner
// has successfully built the cluster.
func TestRawKeepsAControlPlaneForGraders(t *testing.T) {
	prof, err := LoadProfile(repoContent(t), "raw")
	if err != nil {
		t.Fatalf("load raw profile: %v", err)
	}
	if prof.ControlPlane() == nil {
		t.Fatal("raw profile has no control-plane node; kubectl graders cannot address it")
	}
	if got := len(prof.ClusterNodes()); got != 2 {
		t.Errorf("ClusterNodes() = %d, want 2 (cp1 and worker1)", got)
	}
}

// TestStandardProvisioningIsUnchanged pins the refactor that extracted
// kubeNodeProvision: the shipping profile must run the same scripts in the
// same order it always did.
func TestStandardProvisioningIsUnchanged(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())

	prof, err := LoadProfile(repoContent(t), "standard")
	if err != nil {
		t.Fatalf("load standard profile: %v", err)
	}
	prov := fake.New()
	prov.Responder = joinResponder
	m := New(prof, prov, repoContent(t))
	// distributeKubeconfig reads this off the control plane; on a real node
	// kubeadm init writes it.
	prov.Files[m.VMName("cp1")+":"+AdminKubeconfig] = []byte(
		"apiVersion: v1\nclusters:\n- cluster:\n    server: https://192.168.104.11:6443\n")
	if err := m.Up(context.Background(), false); err != nil {
		t.Fatalf("Up: %v", err)
	}

	for _, want := range []string{"kubeadm init", "calico", "metrics-server", "local-path"} {
		if !prov.Ran(want) {
			t.Errorf("standard provisioning no longer runs %q", want)
		}
	}
	// kube-node.sh must reach the control plane before kubeadm init does.
	cpScripts := prov.ScriptsFor(m.VMName("cp1"))
	kubeNode, init := -1, -1
	for i, s := range cpScripts {
		if kubeNode < 0 && strings.Contains(s, "apt-mark hold kubelet") {
			kubeNode = i
		}
		if init < 0 && strings.Contains(s, "kubeadm init") {
			init = i
		}
	}
	if kubeNode < 0 || init < 0 || kubeNode > init {
		t.Errorf("kube-node.sh (%d) must run before kubeadm init (%d) on cp1", kubeNode, init)
	}
}

// joinResponder answers the two queries a kubeadm bring-up makes against the
// control plane and cannot be faked by returning empty output: the worker join
// command, and the address discovery the default responder already handles.
func joinResponder(node, script string) (provider.ExecResult, error) {
	switch {
	case strings.Contains(script, "kubeadm token create"):
		return provider.ExecResult{Stdout: "kubeadm join 192.168.104.11:6443 --token abcdef.0123456789abcdef " +
			"--discovery-token-ca-cert-hash sha256:" + strings.Repeat("a", 64) + "\n"}, nil
	case strings.Contains(script, "ip -4 -o addr show"):
		return provider.ExecResult{Stdout: "192.168.104." + fmt.Sprint(10+len(node)%40) + "\n"}, nil
	case strings.Contains(script, "test -f"):
		return provider.ExecResult{Stdout: "no\n"}, nil
	case strings.Contains(script, "'get' 'nodes'"):
		// WaitReady counts Ready lines. KubectlRaw shell-quotes every
		// argument, so match the quoted form rather than the plain one.
		return provider.ExecResult{Stdout: "cp1 Ready control-plane\nworker1 Ready <none>\nworker2 Ready <none>\n"}, nil
	default:
		return provider.ExecResult{}, nil
	}
}

func ranOn(prov *fake.Provider, vm, substr string) bool {
	for _, s := range prov.ScriptsFor(vm) {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// TestStandardControlPlaneScriptIsUnchanged is the backward-compatibility guard
// on the one change in the HA work that can break the shipping profile, and it
// breaks it at provision time rather than at compile time: the `{{- if}}` added
// to control-plane.sh must leave a profile with no API endpoint rendering
// exactly as it did before.
func TestStandardControlPlaneScriptIsUnchanged(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())

	prof, err := LoadProfile(repoContent(t), "standard")
	if err != nil {
		t.Fatalf("load standard profile: %v", err)
	}
	prov := fake.New()
	prov.Responder = joinResponder
	m := New(prof, prov, repoContent(t))
	v, err := m.vars(context.Background(), "cp1")
	if err != nil {
		t.Fatalf("vars: %v", err)
	}
	body, err := m.renderScript("control-plane.sh", v)
	if err != nil {
		t.Fatalf("render control-plane.sh: %v", err)
	}

	for _, forbidden := range []string{"--control-plane-endpoint", "--upload-certs"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("standard renders %s; it has no API endpoint", forbidden)
		}
	}
	// A dangling backslash on an otherwise blank line is what a mis-trimmed
	// template produces, and it turns kubeadm init into a syntax error.
	for i, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == `\` {
			t.Errorf("line %d is a dangling line continuation: %q", i+1, line)
		}
	}
	if !strings.Contains(body, `--apiserver-advertise-address=`) {
		t.Error("the advertise address flag went missing")
	}
}

// TestHAControlPlaneScriptCarriesTheEndpoint is the other half: a profile that
// declares one must pass it, because it cannot be added after init.
func TestHAControlPlaneScriptCarriesTheEndpoint(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())

	prof := minimalProfile()
	prof.Nodes = append(prof.Nodes,
		NodeConfig{Name: "cp2", Role: RoleControlPlane, CPUs: 2, Memory: "2GiB", Disk: "24GiB"})
	prof.APIEndpoint.Node = "terminal"
	prof.APIEndpoint.Name = "k8s-api"
	prof.APIEndpoint.Port = 6443
	if err := prof.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	prov := fake.New()
	prov.Responder = joinResponder
	m := New(prof, prov, repoContent(t))
	v, err := m.vars(context.Background(), "cp1")
	if err != nil {
		t.Fatalf("vars: %v", err)
	}
	body, err := m.renderScript("control-plane.sh", v)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{`--control-plane-endpoint="k8s-api:6443"`, "--upload-certs"} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered script is missing %s", want)
		}
	}
}

// TestMultiControlPlaneRequiresAnEndpoint catches the failure that would
// otherwise surface twenty minutes into a provision, as a join error that says
// nothing about the cause.
func TestMultiControlPlaneRequiresAnEndpoint(t *testing.T) {
	prof := minimalProfile()
	prof.Nodes = append(prof.Nodes,
		NodeConfig{Name: "cp2", Role: RoleControlPlane, CPUs: 2, Memory: "2GiB", Disk: "24GiB"})

	err := prof.Validate()
	if err == nil {
		t.Fatal("two control planes with no apiEndpoint validated")
	}
	if !strings.Contains(err.Error(), "apiEndpoint") {
		t.Errorf("error does not name the missing field: %v", err)
	}
}

// TestAPIEndpointMustNotBeAControlPlane guards against a load balancer that is
// also one of its own backends.
func TestAPIEndpointMustNotBeAControlPlane(t *testing.T) {
	prof := minimalProfile()
	prof.APIEndpoint.Node = "cp1"
	prof.APIEndpoint.Name = "k8s-api"
	prof.APIEndpoint.Port = 6443

	err := prof.Validate()
	if err == nil {
		t.Fatal("a control plane was accepted as the load balancer node")
	}
	if !strings.Contains(err.Error(), "backends") {
		t.Errorf("unhelpful error: %v", err)
	}
}
