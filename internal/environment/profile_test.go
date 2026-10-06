package environment

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

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

func TestStandardProfileLoads(t *testing.T) {
	p, err := LoadProfile(repoContent(t), "standard")
	if err != nil {
		t.Fatalf("load standard: %v", err)
	}
	if got := p.Minor(); got != "1.35" {
		t.Errorf("minor = %q, want 1.35", got)
	}
	if p.ControlPlane() == nil {
		t.Error("standard profile has no control plane")
	}
	if p.Workstation() == nil {
		t.Error("standard profile has no workstation; node troubleshooting labs need one")
	}
	if n := len(p.NodesByRole(RoleWorker)); n != 2 {
		t.Errorf("workers = %d, want 2", n)
	}
}

type countProfileReads struct{ count int }

func (f *countProfileReads) Open(name string) (fs.File, error) {
	f.count++
	return nil, fs.ErrNotExist
}

func TestLoadProfileRejectsTraversalBeforeReading(t *testing.T) {
	for _, id := range []string{"../outside", "../../outside", ".", "..", "/absolute", `nested\profile`} {
		reads := &countProfileReads{}
		src := &content.Source{FS: reads}
		if _, err := LoadProfile(src, id); err == nil {
			t.Errorf("LoadProfile(%q) accepted an unsafe selector", id)
		}
		if reads.count != 0 {
			t.Errorf("LoadProfile(%q) read content before rejecting it", id)
		}
	}
}

func TestLoadProfileRejectsMismatchedID(t *testing.T) {
	for _, id := range []string{"other", "../../outside"} {
		src := &content.Source{FS: fstest.MapFS{
			"environments/standard/environment.yaml": &fstest.MapFile{
				Data: []byte("schemaVersion: 1\nid: " + id + "\n"),
			},
		}}
		if _, err := LoadProfile(src, "standard"); err == nil || !strings.Contains(err.Error(), "does not match directory") {
			t.Errorf("manifest id %q: err = %v, want mismatch error", id, err)
		}
	}
}

func TestValidateRejectsUnsafeProfileAndNodeNames(t *testing.T) {
	for _, name := range []string{"../../outside", ".", "..", "two words", "node\nEOF", "$(command)"} {
		t.Run(name, func(t *testing.T) {
			p := minimalProfile()
			p.ID = name
			if err := p.Validate(); err == nil {
				t.Error("Validate accepted an unsafe profile id")
			}
			p = minimalProfile()
			p.Nodes[0].Name = name
			if err := p.Validate(); err == nil {
				t.Error("Validate accepted an unsafe node name")
			}
		})
	}
}

func TestValidateRejectsOverlappingPodSubnet(t *testing.T) {
	// The pod CIDR sharing a prefix with the node network is the mistake that
	// would otherwise surface as unroutable pods twenty minutes into a build.
	p := minimalProfile()
	p.Kubernetes.PodSubnet = "192.168.104.0/16"
	err := p.Validate()
	if err == nil || !strings.Contains(err.Error(), "overlaps") {
		t.Fatalf("err = %v, want an overlap complaint", err)
	}
}

func TestValidateRejectsFloatingVersion(t *testing.T) {
	p := minimalProfile()
	p.Kubernetes.Version = "latest"
	err := p.Validate()
	if err == nil || !strings.Contains(err.Error(), "exact patch") {
		t.Fatalf("err = %v, want a complaint about pinning", err)
	}
}

func TestValidateRequiresControlPlaneForKubeadm(t *testing.T) {
	p := minimalProfile()
	p.Nodes = p.Nodes[:1] // workstation only
	err := p.Validate()
	if err == nil || !strings.Contains(err.Error(), "control-plane") {
		t.Fatalf("err = %v, want a complaint about the control plane", err)
	}
}

func TestSubnetPrefix(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"192.168.104.0/24", "192.168.104."},
		{"10.0.0.0/8", "10.0.0."},
	} {
		if got := subnetPrefix(tc.in); got != tc.want {
			t.Errorf("subnetPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRewriteServer(t *testing.T) {
	in := "clusters:\n- cluster:\n    server: https://127.0.0.1:6443\n    x: y\n"
	got := rewriteServer(in, "https://192.168.104.5:6443")
	if !strings.Contains(got, "    server: https://192.168.104.5:6443") {
		t.Errorf("server not rewritten:\n%s", got)
	}
	if !strings.Contains(got, "    x: y") {
		t.Errorf("other lines were disturbed:\n%s", got)
	}
}

func minimalProfile() *Profile {
	p := &Profile{SchemaVersion: 1, ID: "test", Provisioning: ProvisionKubeadm}
	p.Kubernetes.Version = "1.35.8"
	p.Kubernetes.PodSubnet = "10.244.0.0/16"
	p.Kubernetes.ServiceSubnet = "10.96.0.0/12"
	p.Network.Lima = "user-v2"
	p.Network.Subnet = "192.168.104.0/24"
	p.Nodes = []NodeConfig{
		{Name: "terminal", Role: RoleWorkstation, CPUs: 2, Memory: "1GiB", Disk: "16GiB"},
		{Name: "cp1", Role: RoleControlPlane, CPUs: 2, Memory: "3GiB", Disk: "24GiB"},
	}
	return p
}
