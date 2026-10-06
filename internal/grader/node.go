package grader

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/environment"
	"github.com/gustavfredrikson/cka-dojo/internal/provider"
)

func init() {
	register(func() Checker { return &NodeService{} })
	register(func() Checker { return &NodeFile{} })
	register(func() Checker { return &Command{} })
	register(func() Checker { return &NodeReady{} })
}

func nowNano() int64 { return time.Now().UnixNano() }

// NodeService checks a systemd unit's state on a node.
type NodeService struct {
	Node string `yaml:"node"`
	Unit string `yaml:"unit"`
	// State is active or inactive; defaults to active.
	State string `yaml:"state"`
	// Enabled, when set, additionally requires the unit to be enabled at boot.
	Enabled *bool `yaml:"enabled"`
}

func (n *NodeService) Type() string { return "nodeService" }

func (n *NodeService) Validate() error {
	if n.Node == "" || n.Unit == "" {
		return fmt.Errorf("node and unit are required")
	}
	switch n.State {
	case "", "active", "inactive":
	default:
		return fmt.Errorf("state must be active or inactive, got %q", n.State)
	}
	return nil
}

func (n *NodeService) want() string {
	if n.State == "" {
		return "active"
	}
	return n.State
}

func (n *NodeService) Describe() string {
	s := fmt.Sprintf("%s on %s is %s", n.Unit, n.Node, n.want())
	if n.Enabled != nil {
		if *n.Enabled {
			s += " and enabled at boot"
		} else {
			s += " and disabled at boot"
		}
	}
	return s
}

func (n *NodeService) Check(ctx context.Context, env *environment.Manager) Result {
	script := fmt.Sprintf("systemctl is-active %[1]s || true\nsystemctl is-enabled %[1]s 2>/dev/null || true\n", shellWord(n.Unit))
	res, err := env.Exec(ctx, n.Node, script, "root")
	if err != nil {
		return broken(n.Describe(), err)
	}
	lines := nonEmptyLines(res.Stdout)
	active, enabled := "", ""
	if len(lines) > 0 {
		active = strings.TrimSpace(lines[0])
	}
	if len(lines) > 1 {
		enabled = strings.TrimSpace(lines[1])
	}
	if active != n.want() {
		return fail(n.Describe(), "systemctl reports %q", active)
	}
	if n.Enabled != nil {
		isEnabled := enabled == "enabled" || enabled == "enabled-runtime"
		if isEnabled != *n.Enabled {
			return fail(n.Describe(), "systemctl reports it %q at boot", enabled)
		}
	}
	return pass(n.Describe())
}

// NodeFile checks a file on a node: that it exists, and optionally what it
// contains.
type NodeFile struct {
	Node string `yaml:"node"`
	Path string `yaml:"path"`
	// Absent inverts the existence check.
	Absent bool `yaml:"absent"`
	Match  `yaml:",inline"`
}

func (n *NodeFile) Type() string { return "nodeFile" }

func (n *NodeFile) Validate() error {
	if n.Node == "" || n.Path == "" {
		return fmt.Errorf("node and path are required")
	}
	if n.Absent && !n.Match.empty() {
		return fmt.Errorf("absent cannot be combined with a content match")
	}
	return n.Match.validate()
}

func (n *NodeFile) Describe() string {
	switch {
	case n.Absent:
		return fmt.Sprintf("%s does not exist on %s", n.Path, n.Node)
	case n.Match.empty():
		return fmt.Sprintf("%s exists on %s", n.Path, n.Node)
	default:
		return fmt.Sprintf("%s on %s %s", n.Path, n.Node, n.Match.describe())
	}
}

func (n *NodeFile) Check(ctx context.Context, env *environment.Manager) Result {
	script := fmt.Sprintf("if [ -e %[1]s ]; then echo PRESENT; cat %[1]s 2>/dev/null || true; else echo ABSENT; fi\n", shellWord(n.Path))
	res, err := env.Exec(ctx, n.Node, script, "root")
	if err != nil {
		return broken(n.Describe(), err)
	}
	out := res.Stdout
	present := strings.HasPrefix(out, "PRESENT")
	body := ""
	if i := strings.Index(out, "\n"); i >= 0 {
		body = out[i+1:]
	}
	if n.Absent {
		if present {
			return fail(n.Describe(), "the file is still there")
		}
		return pass(n.Describe())
	}
	if !present {
		return fail(n.Describe(), "the file does not exist")
	}
	if n.Match.empty() || n.Match.ok(body) {
		return pass(n.Describe())
	}
	return fail(n.Describe(), "the contents do not match")
}

// NodeReady checks a Kubernetes node's Ready condition.
type NodeReady struct {
	Node string `yaml:"node"`
	// Ready defaults to true.
	Ready *bool `yaml:"ready"`
	// Schedulable, when set, checks the node is not cordoned.
	Schedulable *bool `yaml:"schedulable"`
}

func (n *NodeReady) Type() string { return "nodeReady" }

func (n *NodeReady) Validate() error {
	if n.Node == "" {
		return fmt.Errorf("node is required")
	}
	return nil
}

func (n *NodeReady) wantReady() bool { return n.Ready == nil || *n.Ready }

func (n *NodeReady) Describe() string {
	s := fmt.Sprintf("node %s is", n.Node)
	if !n.wantReady() {
		s += " not"
	}
	s += " Ready"
	if n.Schedulable != nil {
		if *n.Schedulable {
			s += " and schedulable"
		} else {
			s += " and cordoned"
		}
	}
	return s
}

func (n *NodeReady) Check(ctx context.Context, env *environment.Manager) Result {
	res, err := env.KubectlRaw(ctx, "get", "node", n.Node,
		"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}"+fieldSep+"{.spec.unschedulable}")
	if err != nil {
		return broken(n.Describe(), err)
	}
	if res.ExitCode != 0 {
		return fail(n.Describe(), "the node is not registered with the API server")
	}
	fields := splitFields(res.Stdout, 2)
	ready := fields[0] == "True"
	if ready != n.wantReady() {
		state := "NotReady"
		if ready {
			state = "Ready"
		}
		return fail(n.Describe(), "the node is %s", state)
	}
	if n.Schedulable != nil {
		unschedulable := fields[1] == "true"
		if unschedulable == *n.Schedulable {
			if unschedulable {
				return fail(n.Describe(), "the node is cordoned")
			}
			return fail(n.Describe(), "the node is schedulable")
		}
	}
	return pass(n.Describe())
}

// Command is the escape hatch for checks the named graders do not cover. It
// runs on a node, or on the control plane when target is empty.
type Command struct {
	// Target is a node name, or empty for the control plane.
	Target  string `yaml:"target"`
	Command string `yaml:"command"`
	// ExitCode defaults to 0.
	ExitCode *int `yaml:"exitCode"`
	// Stdout, when set, is matched against the command's output.
	Stdout *Match `yaml:"stdout"`
	// Description overrides the generated requirement text, so a lab can
	// avoid leaking the answer through the check itself.
	Description string `yaml:"description"`
}

func (c *Command) Type() string { return "command" }

func (c *Command) Validate() error {
	if strings.TrimSpace(c.Command) == "" {
		return fmt.Errorf("command is required")
	}
	if c.Stdout != nil {
		return c.Stdout.validate()
	}
	return nil
}

func (c *Command) Describe() string {
	if c.Description != "" {
		return c.Description
	}
	where := c.Target
	if where == "" {
		where = "the control plane"
	}
	if c.Stdout != nil {
		return fmt.Sprintf("on %s, `%s` output %s", where, c.Command, c.Stdout.describe())
	}
	return fmt.Sprintf("on %s, `%s` succeeds", where, c.Command)
}

func (c *Command) Check(ctx context.Context, env *environment.Manager) Result {
	node := c.Target
	if node == "" {
		cp := env.Profile.ControlPlane()
		if cp == nil {
			return broken(c.Describe(), fmt.Errorf("no control plane in profile %s", env.Profile.ID))
		}
		node = cp.Name
	}
	script := fmt.Sprintf("export KUBECONFIG=%s\n%s\n", environment.AdminKubeconfig, c.Command)
	res, err := env.Exec(ctx, node, script, "root")
	if ctxErr := ctx.Err(); ctxErr != nil {
		return broken(c.Describe(), ctxErr)
	}
	if err != nil {
		// A command's non-zero exit may be the expected answer. Failure to
		// execute it has no answer, even when the result's default exit code
		// happens to match the requirement.
		var exitErr *provider.ExitError
		if !errors.As(err, &exitErr) {
			return broken(c.Describe(), err)
		}
	}
	want := 0
	if c.ExitCode != nil {
		want = *c.ExitCode
	}
	if res.ExitCode != want {
		return fail(c.Describe(), "exit code %d (wanted %d): %s", res.ExitCode, want, firstLine(res.Stderr+res.Stdout))
	}
	if c.Stdout != nil && !c.Stdout.ok(res.Stdout) {
		return fail(c.Describe(), "output was %q", strings.TrimSpace(firstLine(res.Stdout)))
	}
	return pass(c.Describe())
}

func shellWord(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
