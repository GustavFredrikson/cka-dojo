package lima

import (
	"strings"
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/provider"
)

func TestInstanceYAML(t *testing.T) {
	got := instanceYAML(provider.NodeSpec{
		Name: "cka-dojo-standard-cp1", Hostname: "cp1",
		CPUs: 2, Memory: "3GiB", Disk: "24GiB", Network: "user-v2",
	})

	// The training repository must never be reachable from inside a guest,
	// or a learner can read the solution to the lab they are sitting in.
	if !strings.Contains(got, "mounts: []") {
		t.Error("instance has host mounts")
	}
	// We install containerd ourselves, because labs break it on purpose.
	if !strings.Contains(got, "containerd:\n  system: false\n  user: false") {
		t.Error("Lima's own containerd was left enabled")
	}
	// Kubernetes node names come from the hostname, so it has to be right on
	// first boot rather than set later.
	if !strings.Contains(got, "hostnamectl set-hostname cp1") {
		t.Error("hostname is not set during first boot")
	}
	if !strings.Contains(got, "- lima: user-v2") {
		t.Error("the node is not on the environment network")
	}
	for _, want := range []string{`memory: "3GiB"`, `disk: "24GiB"`, "cpus: 2"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestInstanceYAMLWithoutNetwork(t *testing.T) {
	got := instanceYAML(provider.NodeSpec{Name: "x", CPUs: 1, Memory: "1GiB", Disk: "8GiB"})
	if strings.Contains(got, "networks:") {
		t.Error("a network was configured when none was asked for")
	}
}

func TestMapStatus(t *testing.T) {
	for in, want := range map[string]provider.Status{
		"Running": provider.StatusRunning,
		"Stopped": provider.StatusStopped,
		"Broken":  provider.StatusBroken,
		"":        provider.StatusUnknown,
	} {
		if got := mapStatus(in); got != want {
			t.Errorf("mapStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLastLines(t *testing.T) {
	if got := lastLines("a\nb\nc\nd\n", 2); got != "c; d" {
		t.Errorf("lastLines = %q", got)
	}
}

// TestWrapScriptSurvivesStdinConsumers is the regression guard for a bug that
// silently truncated scripts: piping a script into `bash -s` means any command
// inside it that reads stdin (ssh, read, an apt prompt) eats the remainder,
// and the tail never runs with no error to show for it.
func TestWrapScriptSurvivesStdinConsumers(t *testing.T) {
	got := wrapScript("ssh worker1 hostname\necho after\n")

	if strings.Contains(got, "ssh worker1") {
		t.Error("the payload is inline, so a stdin consumer can still eat it")
	}
	if !strings.Contains(got, `bash "$f" </dev/null`) {
		t.Error("the payload does not run with stdin closed")
	}
	if !strings.Contains(got, "rc=$?") || !strings.Contains(got, "exit $rc") {
		t.Error("the payload's exit code is not propagated")
	}
	if !strings.Contains(got, `rm -f "$f"`) {
		t.Error("the temporary script is left behind in the guest")
	}
}

func TestWrapScriptEncodesAwkwardCharacters(t *testing.T) {
	// Single quotes, backticks and heredoc-looking text must all survive.
	script := "echo 'it''s' `date` <<'EOF'\nEOF\n"
	got := wrapScript(script)
	for _, forbidden := range []string{"`date`", "it''s"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("%q leaked into the wrapper unencoded", forbidden)
		}
	}
}
