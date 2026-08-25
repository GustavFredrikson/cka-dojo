package fault

import (
	"context"
	"fmt"
	"io/fs"
	"strings"

	"github.com/gustavfredrikson/cka-dojo/internal/environment"
)

func init() {
	register(func() Fault { return &SystemdStop{} })
	register(func() Fault { return &FileReplace{} })
	register(func() Fault { return &NodeExec{} })
}

// backupDir is where node faults stash originals so Repair can restore them.
const backupDir = "/var/lib/dojo/backup"

// SystemdStop stops a unit on a node, optionally disabling it so a reboot
// does not quietly fix the lab.
type SystemdStop struct {
	Node string `yaml:"node"`
	Unit string `yaml:"unit"`
	// Disable also masks the unit at boot.
	Disable bool `yaml:"disable"`
}

func (s *SystemdStop) Type() string { return "systemdStop" }

func (s *SystemdStop) Validate() error {
	if s.Node == "" {
		return fmt.Errorf("node is required")
	}
	if s.Unit == "" {
		return fmt.Errorf("unit is required")
	}
	return nil
}

func (s *SystemdStop) Describe() string {
	if s.Disable {
		return fmt.Sprintf("stop and disable %s on %s", s.Unit, s.Node)
	}
	return fmt.Sprintf("stop %s on %s", s.Unit, s.Node)
}

func (s *SystemdStop) Inject(ctx context.Context, env *environment.Manager, lc *Context) error {
	script := fmt.Sprintf("set -euo pipefail\nsystemctl stop %s\n", shellWord(s.Unit))
	if s.Disable {
		script += fmt.Sprintf("systemctl disable %s >/dev/null 2>&1 || true\n", shellWord(s.Unit))
	}
	_, err := env.Run(ctx, s.Node, script)
	return err
}

func (s *SystemdStop) Repair(ctx context.Context, env *environment.Manager, lc *Context) error {
	script := fmt.Sprintf(`set -euo pipefail
systemctl enable %[1]s >/dev/null 2>&1 || true
systemctl start %[1]s
`, shellWord(s.Unit))
	_, err := env.Run(ctx, s.Node, script)
	return err
}

// FileReplace overwrites a file on a node with a body from the lab directory,
// keeping the original so Repair can put it back.
type FileReplace struct {
	Node string `yaml:"node"`
	Path string `yaml:"path"`
	// Source is a file in the lab directory.
	Source string `yaml:"source"`
	// Content is an inline alternative to Source.
	Content string `yaml:"content"`
	// Restart is a unit to restart after the file changes.
	Restart string `yaml:"restart"`
}

func (f *FileReplace) Type() string { return "fileReplace" }

func (f *FileReplace) Validate() error {
	if f.Node == "" {
		return fmt.Errorf("node is required")
	}
	if f.Path == "" {
		return fmt.Errorf("path is required")
	}
	if (f.Source == "") == (f.Content == "") {
		return fmt.Errorf("exactly one of source or content is required")
	}
	return nil
}

func (f *FileReplace) Describe() string { return fmt.Sprintf("replace %s on %s", f.Path, f.Node) }

// backupPath is a stable, filesystem-safe name for one replaced file.
func (f *FileReplace) backupPath() string {
	return backupDir + "/" + strings.ReplaceAll(strings.TrimPrefix(f.Path, "/"), "/", "_")
}

func (f *FileReplace) Inject(ctx context.Context, env *environment.Manager, lc *Context) error {
	body := []byte(f.Content)
	if f.Source != "" {
		var err error
		body, err = fs.ReadFile(lc.Files, f.Source)
		if err != nil {
			return fmt.Errorf("read %s: %w", f.Source, err)
		}
	}
	backup := f.backupPath()
	pre := fmt.Sprintf(`set -euo pipefail
mkdir -p %s
if [ ! -f %s ] && [ -f %s ]; then cp -a %s %s; fi
`, shellWord(backupDir), shellWord(backup), shellWord(f.Path), shellWord(f.Path), shellWord(backup))
	if _, err := env.Run(ctx, f.Node, pre); err != nil {
		return err
	}
	if err := env.Prov.WriteFile(ctx, env.VMName(f.Node), f.Path, body, 0o644); err != nil {
		return err
	}
	return f.restart(ctx, env)
}

func (f *FileReplace) Repair(ctx context.Context, env *environment.Manager, lc *Context) error {
	script := fmt.Sprintf(`set -euo pipefail
if [ -f %s ]; then cp -a %s %s; rm -f %s; fi
`, shellWord(f.backupPath()), shellWord(f.backupPath()), shellWord(f.Path), shellWord(f.backupPath()))
	if _, err := env.Run(ctx, f.Node, script); err != nil {
		return err
	}
	return f.restart(ctx, env)
}

func (f *FileReplace) restart(ctx context.Context, env *environment.Manager) error {
	if f.Restart == "" {
		return nil
	}
	_, err := env.Run(ctx, f.Node, fmt.Sprintf("systemctl restart %s\n", shellWord(f.Restart)))
	return err
}

// NodeExec is the escape hatch: an arbitrary script on a node, with an
// author-supplied undo. Prefer a named primitive when one fits.
type NodeExec struct {
	Node   string `yaml:"node"`
	Script string `yaml:"script"`
	Undo   string `yaml:"undo"`
}

func (n *NodeExec) Type() string { return "nodeExec" }

func (n *NodeExec) Validate() error {
	if n.Node == "" {
		return fmt.Errorf("node is required")
	}
	if strings.TrimSpace(n.Script) == "" {
		return fmt.Errorf("script is required")
	}
	if strings.TrimSpace(n.Undo) == "" {
		return fmt.Errorf("undo is required so `dojo reset` can put the node back")
	}
	return nil
}

func (n *NodeExec) Describe() string { return "run a script on " + n.Node }

func (n *NodeExec) Inject(ctx context.Context, env *environment.Manager, lc *Context) error {
	_, err := env.Run(ctx, n.Node, "set -euo pipefail\n"+n.Script)
	return err
}

func (n *NodeExec) Repair(ctx context.Context, env *environment.Manager, lc *Context) error {
	_, err := env.Run(ctx, n.Node, "set -euo pipefail\n"+n.Undo)
	return err
}

// shellWord quotes a value for safe use as one shell word.
func shellWord(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
