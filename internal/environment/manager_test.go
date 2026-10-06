package environment

import (
	"context"
	"testing"

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
