package lab

import (
	"context"
	"fmt"
	"io/fs"
	"strings"

	"github.com/gustavfredrikson/cka-dojo/internal/environment"
	"github.com/gustavfredrikson/cka-dojo/internal/fault"
	"github.com/gustavfredrikson/cka-dojo/internal/grader"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
)

// Runner executes one lab plan against one environment.
type Runner struct {
	Env  *environment.Manager
	Plan *Plan
}

// NewRunner pairs a plan with an environment.
func NewRunner(env *environment.Manager, plan *Plan) *Runner {
	return &Runner{Env: env, Plan: plan}
}

func (r *Runner) faultContext() *fault.Context {
	return &fault.Context{Files: r.Plan.Lab.Files, LabID: r.Plan.Lab.ID}
}

// Setup builds the scenario: baseline first, then the faults, so the learner
// meets a cluster that was working and then broke.
func (r *Runner) Setup(ctx context.Context) error {
	for _, manifest := range r.Plan.Manifests {
		data, err := fs.ReadFile(r.Plan.Lab.Files, manifest)
		if err != nil {
			return fmt.Errorf("read %s: %w", manifest, err)
		}
		ui.Detail("applying %s", manifest)
		if err := r.Env.Apply(ctx, data); err != nil {
			return fmt.Errorf("apply %s: %w", manifest, err)
		}
	}
	if err := r.waitReady(ctx); err != nil {
		return err
	}
	for _, f := range r.Plan.Faults {
		ui.Detail("injecting fault: %s", f.Describe())
		if err := f.Inject(ctx, r.Env, r.faultContext()); err != nil {
			return fmt.Errorf("inject %s: %w", f.Type(), err)
		}
	}
	return nil
}

// waitReady lets the baseline settle before faults land, so the learner does
// not diagnose a workload that was merely still starting.
func (r *Runner) waitReady(ctx context.Context) error {
	for _, w := range r.Plan.Lab.Setup.WaitReady {
		kind := w.Kind
		if kind == "" {
			kind = "deployment"
		}
		args := []string{"rollout", "status", kind + "/" + w.Name, "--timeout=3m"}
		if w.Namespace != "" {
			args = append(args, "-n", w.Namespace)
		}
		ui.Detail("waiting for %s/%s", kind, w.Name)
		if _, err := r.Env.Kubectl(ctx, args...); err != nil {
			return fmt.Errorf("waiting for %s/%s: %w", kind, w.Name, err)
		}
	}
	return nil
}

// Teardown removes the scenario. Faults are repaired first, because a stopped
// kubelet would stop the object deletions from completing.
func (r *Runner) Teardown(ctx context.Context) error {
	var firstErr error
	for i := len(r.Plan.Faults) - 1; i >= 0; i-- {
		f := r.Plan.Faults[i]
		ui.Detail("repairing fault: %s", f.Describe())
		if err := f.Repair(ctx, r.Env, r.faultContext()); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("repair %s: %w", f.Type(), err)
		}
	}
	for i := len(r.Plan.Manifests) - 1; i >= 0; i-- {
		manifest := r.Plan.Manifests[i]
		data, err := fs.ReadFile(r.Plan.Lab.Files, manifest)
		if err != nil {
			continue
		}
		ui.Detail("removing %s", manifest)
		if err := r.Env.DeleteManifest(ctx, data); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("remove %s: %w", manifest, err)
		}
	}
	return firstErr
}

// Reset puts the lab back to its starting state. It handles the normal case
// cheaply; `dojo env reset` remains the escape hatch for a cluster a learner
// has damaged in some way a lab cannot anticipate.
func (r *Runner) Reset(ctx context.Context) error {
	if err := r.Teardown(ctx); err != nil {
		ui.Warn("cleanup was incomplete: %v", err)
		ui.Info("   if the lab still misbehaves, run `dojo env reset`")
	}
	if err := r.waitGone(ctx); err != nil {
		ui.Detail("proceeding despite lingering objects: %v", err)
	}
	return r.Setup(ctx)
}

// waitGone gives namespace deletion a moment to finish, since re-applying
// into a terminating namespace fails.
func (r *Runner) waitGone(ctx context.Context) error {
	namespaces := map[string]bool{}
	for _, manifest := range r.Plan.Manifests {
		data, err := fs.ReadFile(r.Plan.Lab.Files, manifest)
		if err != nil {
			continue
		}
		for _, name := range namespaceNames(string(data)) {
			namespaces[name] = true
		}
	}
	for name := range namespaces {
		script := fmt.Sprintf(`export KUBECONFIG=%s
for i in $(seq 1 60); do
  kubectl get namespace %s >/dev/null 2>&1 || exit 0
  sleep 2
done
exit 1
`, environment.AdminKubeconfig, name)
		cp := r.Env.Profile.ControlPlane()
		if cp == nil {
			return nil
		}
		if _, err := r.Env.Run(ctx, cp.Name, script); err != nil {
			return fmt.Errorf("namespace %s is still terminating", name)
		}
	}
	return nil
}

// namespaceNames scrapes Namespace object names out of a manifest. A YAML
// round-trip would be more correct, but lab baselines are hand-written and
// this keeps the dependency surface small.
func namespaceNames(manifest string) []string {
	var out []string
	docs := strings.Split(manifest, "\n---")
	for _, doc := range docs {
		if !strings.Contains(doc, "kind: Namespace") {
			continue
		}
		inMeta := false
		for _, line := range strings.Split(doc, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "metadata:" {
				inMeta = true
				continue
			}
			if inMeta && strings.HasPrefix(trimmed, "name:") {
				out = append(out, strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "name:")), `"'`))
				break
			}
			if inMeta && !strings.HasPrefix(line, " ") && trimmed != "" {
				inMeta = false
			}
		}
	}
	return out
}

// Report is the outcome of grading one attempt.
type Report struct {
	Required []grader.Result
	Any      []grader.Result
	Passed   bool
}

// Grade evaluates every requirement. It runs all checks even after the first
// failure, because a learner deserves the whole picture.
func (r *Runner) Grade(ctx context.Context) (*Report, error) {
	rep := &Report{Passed: true}
	for _, c := range r.Plan.Checks {
		res := c.Check(ctx, r.Env)
		rep.Required = append(rep.Required, res)
		if !res.Passed {
			rep.Passed = false
		}
	}
	if len(r.Plan.AnyChecks) > 0 {
		anyPassed := false
		for _, c := range r.Plan.AnyChecks {
			res := c.Check(ctx, r.Env)
			rep.Any = append(rep.Any, res)
			if res.Passed {
				anyPassed = true
			}
		}
		if !anyPassed {
			rep.Passed = false
		}
	}
	return rep, nil
}
