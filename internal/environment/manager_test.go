package environment

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/config"
	"github.com/gustavfredrikson/cka-dojo/internal/provider"
	"github.com/gustavfredrikson/cka-dojo/internal/provider/fake"
)

// TestStopOthersLeavesThisProfileAlone is the guard on the memory budget: only
// one environment is meant to be up, and the machine that enforces it must not
// shut down the environment the caller is about to use.
func TestStopOthersLeavesThisProfileAlone(t *testing.T) {
	prof := minimalProfile()
	prov := fake.New()
	m := New(prof, prov, repoContent(t))

	prov.Nodes[m.VMName("terminal")] = provider.StatusRunning
	prov.Nodes[m.VMName("cp1")] = provider.StatusRunning
	prov.Nodes[NamePrefix+"ha-cp1"] = provider.StatusRunning
	prov.Nodes[NamePrefix+"ha-cp2"] = provider.StatusStopped
	// A machine that is not ours at all.
	prov.Nodes["some-other-vm"] = provider.StatusRunning

	if err := m.StopOthers(context.Background()); err != nil {
		t.Fatalf("StopOthers: %v", err)
	}

	want := map[string]provider.Status{
		m.VMName("terminal"):  provider.StatusRunning,
		m.VMName("cp1"):       provider.StatusRunning,
		NamePrefix + "ha-cp1": provider.StatusStopped,
		NamePrefix + "ha-cp2": provider.StatusStopped,
		"some-other-vm":       provider.StatusRunning,
	}
	for name, wantStatus := range want {
		got, _ := prov.Status(context.Background(), name)
		if got != wantStatus {
			t.Errorf("%s status = %q, want %q", name, got, wantStatus)
		}
	}
}

func TestDestroyRejectsUnsafeProfilesBeforeDeleting(t *testing.T) {
	for _, attack := range []string{"traversal", "profile symlink", "env symlink", "unsafe node"} {
		t.Run(attack, func(t *testing.T) {
			base := t.TempDir()
			home := filepath.Join(base, "home")
			t.Setenv("DOJO_HOME", home)
			outside := filepath.Join(base, "outside")
			if err := os.MkdirAll(outside, 0o700); err != nil {
				t.Fatal(err)
			}
			sentinel := filepath.Join(outside, "keep")
			if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
				t.Fatal(err)
			}
			p := minimalProfile()
			switch attack {
			case "traversal":
				p.ID = "../../outside"
			case "unsafe node":
				p.Nodes[0].Name = "../../outside"
			case "profile symlink", "env symlink":
				link := filepath.Join(home, "env")
				if attack == "profile symlink" {
					link = filepath.Join(link, p.ID)
				}
				if err := os.MkdirAll(filepath.Dir(link), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, link); err != nil {
					t.Fatal(err)
				}
			}
			prov := fake.New()
			m := New(p, prov, repoContent(t))
			vm := m.VMName("cp1")
			prov.Nodes[vm] = provider.StatusRunning
			if err := m.Destroy(context.Background()); err == nil {
				t.Fatal("Destroy accepted an unsafe environment")
			}
			if _, err := os.Stat(sentinel); err != nil {
				t.Fatalf("Destroy touched an unrelated host file: %v", err)
			}
			if got, _ := prov.Status(context.Background(), vm); got != provider.StatusRunning {
				t.Errorf("Destroy changed a VM before rejecting an unsafe environment: %s", got)
			}
		})
	}
}

func TestDestroyRemovesOnlySelectedEnvironment(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())
	p := minimalProfile()
	prov := fake.New()
	m := New(p, prov, repoContent(t))
	dir, err := config.EnvDir(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	other, err := config.EnvDir("other")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range p.Nodes {
		prov.Nodes[m.VMName(n.Name)] = provider.StatusRunning
	}
	otherVM := NamePrefix + "other-cp1"
	prov.Nodes[otherVM] = provider.StatusRunning
	if err := m.Destroy(context.Background()); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("selected environment artifacts remain: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("Destroy removed another profile's artifacts: %v", err)
	}
	for _, n := range p.Nodes {
		if got, _ := prov.Status(context.Background(), m.VMName(n.Name)); got != provider.StatusMissing {
			t.Errorf("selected VM %s remains: %s", n.Name, got)
		}
	}
	if got, _ := prov.Status(context.Background(), otherVM); got != provider.StatusRunning {
		t.Errorf("Destroy changed another profile's VM: %s", got)
	}
}

func TestDestroyAbsentEnvironmentDoesNotCreateHostDirectories(t *testing.T) {
	home := filepath.Join(t.TempDir(), "missing-home")
	t.Setenv("DOJO_HOME", home)
	m := New(minimalProfile(), fake.New(), repoContent(t))
	if err := m.Destroy(context.Background()); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("Destroy created a host state directory: %v", err)
	}
}

type changeArtifactsOnDestroy struct {
	*fake.Provider
	change func()
}

func (p *changeArtifactsOnDestroy) DestroyNode(ctx context.Context, name string) error {
	if err := p.Provider.DestroyNode(ctx, name); err != nil {
		return err
	}
	if p.change != nil {
		p.change()
		p.change = nil
	}
	return nil
}

func TestDestroyReportsArtifactCleanupFailure(t *testing.T) {
	t.Setenv("DOJO_HOME", t.TempDir())
	prof := minimalProfile()
	dir, err := config.EnvDir(prof.ID)
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "keep")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	prov := &changeArtifactsOnDestroy{Provider: fake.New(), change: func() {
		// Simulate the artifacts changing after the preflight check. Cleanup
		// must check again and surface its error instead of claiming success.
		if err := os.Remove(dir); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, dir); err != nil {
			t.Fatal(err)
		}
	}}
	m := New(prof, prov, repoContent(t))
	if err := m.Destroy(context.Background()); err == nil {
		t.Fatal("Destroy hid an artifact cleanup failure")
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("cleanup followed a replacement symlink: %v", err)
	}
}

// TestProfileSizeTotals pins the arithmetic `dojo doctor` sizes the host with.
// Lima accepts several spellings and the profiles in this repo use MiB as
// freely as GiB, so a parser that silently returned zero for one of them would
// quietly greenlight a host that cannot carry the environment.
func TestProfileSizeTotals(t *testing.T) {
	prof := minimalProfile()
	prof.Nodes = []NodeConfig{
		{Name: "terminal", Role: RoleWorkstation, CPUs: 2, Memory: "1GiB", Disk: "16GiB"},
		{Name: "cp1", Role: RoleControlPlane, CPUs: 2, Memory: "3GiB", Disk: "24GiB"},
		{Name: "worker1", Role: RoleWorker, CPUs: 2, Memory: "2560MiB", Disk: "24GiB"},
	}

	if got, want := prof.TotalMemoryGiB(), 6.5; got != want {
		t.Errorf("TotalMemoryGiB() = %v, want %v", got, want)
	}
	if got, want := prof.TotalDiskGiB(), 64.0; got != want {
		t.Errorf("TotalDiskGiB() = %v, want %v", got, want)
	}
}

func TestSizeGiB(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"1GiB", 1},
		{"2500MiB", 2500.0 / 1024},
		{"4G", 4},
		{"512M", 0.5},
		{"", 0},
		{"garbage", 0},
		{"1073741824", 1},
	}
	for _, c := range cases {
		if got := sizeGiB(c.in); got != c.want {
			t.Errorf("sizeGiB(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
