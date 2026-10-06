package cli

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/gustavfredrikson/cka-dojo/internal/environment"
	"github.com/gustavfredrikson/cka-dojo/internal/provider/lima"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
	"github.com/spf13/cobra"
)

// check is one line of doctor output.
type check struct {
	ok      bool
	msg     string
	remedy  string
	warning bool
}

func newDoctorCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check that this machine can run a dojo environment",
		Long: `doctor verifies the host, the dependencies and the content before you
spend twenty minutes provisioning a cluster that was never going to work.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd.Context(), app)
		},
	}
}

func runDoctor(ctx context.Context, app *App) error {
	failures := 0

	section := func(name string) { ui.Blank(); ui.Heading("%s", name) }
	report := func(c check) {
		switch {
		case c.ok:
			ui.OK("%s", c.msg)
		case c.warning:
			ui.Warn("%s", c.msg)
			if c.remedy != "" {
				ui.Info("   %s", c.remedy)
			}
		default:
			failures++
			ui.Fail("%s", c.msg)
			if c.remedy != "" {
				ui.Info("   %s", c.remedy)
			}
		}
	}

	section("Host")
	report(check{
		ok:      runtime.GOOS == "darwin",
		msg:     fmt.Sprintf("operating system: %s", runtime.GOOS),
		warning: true,
		remedy:  "only macOS is exercised today; other hosts may work but are untested",
	})
	report(check{
		ok:      runtime.GOARCH == "arm64",
		msg:     fmt.Sprintf("architecture: %s", runtime.GOARCH),
		warning: true,
		remedy:  "only arm64 (Apple Silicon) is exercised today",
	})

	// Size the host against the profile that is actually selected. Hardcoding
	// the standard profile's numbers would greenlight a heavier one on a host
	// that cannot carry it.
	profileID := app.Profile()
	guestGiB, diskGiB := 9.0, 30.0
	wantMem, wantDisk := 16.0, 40.0
	if prof, perr := environment.LoadProfile(app.Src, profileID); perr == nil {
		guestGiB = prof.TotalMemoryGiB()
		diskGiB = prof.TotalDiskGiB()
		// Guests plus headroom for macOS and everything else the user is
		// running. The standard profile's 9 GiB is what put the floor at 16.
		wantMem = guestGiB + 7
		wantDisk = diskGiB * 0.5
	}

	memGiB := hostMemoryGiB()
	report(check{
		ok:  memGiB >= wantMem,
		msg: fmt.Sprintf("memory: %.0f GiB", memGiB),
		remedy: fmt.Sprintf("the %s profile wants about %.0f GiB of guest memory; %.0f GiB of host RAM is the practical floor",
			profileID, guestGiB, wantMem),
	})

	freeGiB, err := freeDiskGiB("/")
	if err == nil {
		report(check{
			ok:  freeGiB >= wantDisk,
			msg: fmt.Sprintf("free disk: %.0f GiB", freeGiB),
			remedy: fmt.Sprintf("the %s profile declares %.0f GiB of thin-provisioned disk and uses well under that once built; keep %.0f GiB free",
				profileID, diskGiB, wantDisk),
		})
	}

	section("Dependencies")
	limaProv := lima.New()
	if err := limaProv.Available(); err != nil {
		report(check{ok: false, msg: "limactl not found", remedy: "brew install lima"})
	} else {
		v, verr := limaProv.Version(ctx)
		if verr != nil {
			report(check{ok: false, msg: "limactl found but not runnable", remedy: verr.Error()})
		} else {
			report(check{ok: true, msg: "limactl " + v})
		}
	}
	for _, bin := range []string{"ssh", "ssh-keygen"} {
		_, err := exec.LookPath(bin)
		report(check{
			ok:     err == nil,
			msg:    bin + " available",
			remedy: "install the Xcode command line tools: xcode-select --install",
		})
	}

	section("Content")
	report(check{ok: true, msg: "content source: " + app.Src.Origin})

	profiles, err := environment.ListProfiles(app.Src)
	if err != nil {
		report(check{ok: false, msg: "cannot list environment profiles", remedy: err.Error()})
	} else {
		report(check{ok: len(profiles) > 0, msg: "environment profiles: " + strings.Join(profiles, ", ")})
	}

	m, err := app.Manager("")
	if err != nil {
		report(check{ok: false, msg: fmt.Sprintf("profile %q", app.Profile()), remedy: err.Error()})
	} else {
		report(check{ok: true, msg: fmt.Sprintf("profile %q: Kubernetes %s, %s, %d nodes",
			m.Profile.ID, m.Profile.Kubernetes.Version, m.Profile.CNI.Provider, len(m.Profile.Nodes))})

		cur, err := loadCurriculum(app)
		if err != nil {
			report(check{ok: false, msg: "curriculum", remedy: err.Error()})
		} else {
			report(check{ok: true, msg: fmt.Sprintf("curriculum %s: %d labs across %d modules",
				cur.ID, cur.LabCount(), len(cur.Modules))})
			if cur.Kubernetes.Minor != m.Profile.Minor() {
				report(check{
					ok:      false,
					warning: true,
					msg: fmt.Sprintf("curriculum targets Kubernetes %s but profile %q installs %s",
						cur.Kubernetes.Minor, m.Profile.ID, m.Profile.Minor()),
					remedy: "align curriculum.yaml and the environment profile",
				})
			}
		}
	}

	section("Environment")
	if m != nil {
		sts, err := m.Status(ctx)
		if err != nil {
			report(check{warning: true, msg: "cannot read environment status", remedy: err.Error()})
		} else {
			running := 0
			for _, s := range sts {
				if s.Status == "running" {
					running++
				}
			}
			switch {
			case running == len(sts):
				report(check{ok: true, msg: fmt.Sprintf("environment %q is running (%d nodes)", m.Profile.ID, running)})
			case running == 0:
				report(check{warning: true, msg: fmt.Sprintf("environment %q is not created yet", m.Profile.ID),
					remedy: "run `dojo setup`"})
			default:
				report(check{warning: true,
					msg:    fmt.Sprintf("environment %q is partly up (%d/%d nodes)", m.Profile.ID, running, len(sts)),
					remedy: "run `dojo setup` to finish, or `dojo env reset` to start over"})
			}
		}
	}

	ui.Blank()
	if failures > 0 {
		return fmt.Errorf("%d check(s) failed", failures)
	}
	ui.OK("ready")
	return nil
}

func hostMemoryGiB() float64 {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0
	}
	return n / (1 << 30)
}

func freeDiskGiB(path string) (float64, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, err
	}
	return float64(st.Bavail) * float64(st.Bsize) / (1 << 30), nil
}
