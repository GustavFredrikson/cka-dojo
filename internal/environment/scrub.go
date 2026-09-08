package environment

import (
	"context"
	"fmt"
	"strings"

	"github.com/gustavfredrikson/cka-dojo/internal/ui"
)

// scrubKeep is everything the student account cannot lose and still work:
// the SSH key and host config behind `ssh cp1`, and the kubeconfig. Anything
// else in the home directory is the learner's own residue.
var scrubKeep = []string{".ssh", ".kube"}

// scrubStale are caches under a kept directory. kubectl rebuilds its
// discovery cache on demand; the kubeconfig beside it must survive.
var scrubStale = []string{".kube/cache", ".kube/http-cache"}

// ScrubOptions controls how far a scrub goes.
type ScrubOptions struct {
	// DryRun reports what would go without removing anything.
	DryRun bool
	// Defaults re-seeds the dojo .bashrc block after the wipe. Without it
	// the account is bare, which is the point: the exam does not come with
	// three weeks of your aliases already in place.
	Defaults bool
}

// Scrub returns the student account on each node to a freshly created state:
// the learner's files and shell customisations go, the distro skeleton comes
// back, and the environment's own plumbing is left alone.
//
// It is deliberately student-scope. Files a lab had you create as root, and
// dojo's own state under /var/lib/dojo, are not touched: removing the latter
// would break `dojo reset`, which restores pre-fault originals from there.
func (m *Manager) Scrub(ctx context.Context, nodes []string, opts ScrubOptions) error {
	defaults := ""
	if opts.Defaults {
		d, err := m.ShellDefaults()
		if err != nil {
			return err
		}
		defaults = d
	}
	script := scrubScript(scrubKeep, scrubStale, defaults, opts)
	for _, node := range nodes {
		if m.Profile.NodeByName(node) == nil {
			return fmt.Errorf("profile %q has no node %q", m.Profile.ID, node)
		}
	}
	for _, node := range nodes {
		ui.Step("%s", node)
		res, err := m.Exec(ctx, node, script, StudentUser)
		if err != nil {
			return fmt.Errorf("scrub %s: %w", node, err)
		}
		// Always shown, not a --verbose detail: this command deletes
		// things, so what it deleted belongs on screen.
		reported := false
		for _, line := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				ui.Info("   %s", line)
				reported = true
			}
		}
		if !reported {
			ui.Info("   already clean")
		}
	}
	return nil
}

// scrubScript is built rather than templated from content because it must
// keep working when the content root is a stale checkout: a scrub that
// silently skipped would leave the learner sitting on old aliases.
func scrubScript(keep, stale []string, defaults string, opts ScrubOptions) string {
	var b strings.Builder
	b.WriteString("set -euo pipefail\n")
	b.WriteString("cd \"$HOME\"\n")
	// dotglob catches the dotfiles that matter here and, unlike `ls -A`,
	// survives a filename with a space in it.
	b.WriteString("shopt -s dotglob nullglob\n")
	fmt.Fprintf(&b, "keep=%s\n", shellQuote(" "+strings.Join(keep, " ")+" "))
	b.WriteString("for entry in *; do\n")
	b.WriteString("  case \"$keep\" in *\" $entry \"*) continue ;; esac\n")
	// A dotfile the skeleton also ships comes back as the distro wrote it,
	// so call that a reset; anything else is simply gone.
	b.WriteString("  if [ -f \"/etc/skel/$entry\" ]; then\n")
	b.WriteString("    printf 'reset ~/%s\\n' \"$entry\"\n")
	b.WriteString("  else\n")
	b.WriteString("    printf 'remove ~/%s\\n' \"$entry\"\n")
	b.WriteString("  fi\n")
	if !opts.DryRun {
		b.WriteString("  rm -rf -- \"$entry\"\n")
	}
	b.WriteString("done\n")
	for _, s := range stale {
		fmt.Fprintf(&b, "if [ -e %[1]s ]; then\n  printf 'remove ~/%%s\\n' %[1]s\n", shellQuote(s))
		if !opts.DryRun {
			fmt.Fprintf(&b, "  rm -rf -- %s\n", shellQuote(s))
		}
		b.WriteString("fi\n")
	}
	// A fresh account, not a stripped one. Silent: the removal loop above
	// already reported these as resets.
	if !opts.DryRun {
		b.WriteString("for skel in /etc/skel/.[!.]*; do\n")
		b.WriteString("  [ -f \"$skel\" ] || continue\n")
		b.WriteString("  install -m 0644 \"$skel\" \"$HOME/$(basename \"$skel\")\"\n")
		b.WriteString("done\n")
	}
	if defaults != "" && !opts.DryRun {
		b.WriteString("printf 're-seed %s\\n' " + shellQuote(ShellDefaultsMarker) + "\n")
		b.WriteString("cat >> \"$HOME/.bashrc\" <<'DOJO_DEFAULTS_EOF'\n\n")
		b.WriteString(defaults)
		b.WriteString("\nDOJO_DEFAULTS_EOF\n")
	}
	return b.String()
}
