package lab

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/gustavfredrikson/cka-dojo/internal/environment"
	"github.com/gustavfredrikson/cka-dojo/internal/fault"
	"github.com/gustavfredrikson/cka-dojo/internal/grader"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
	"gopkg.in/yaml.v3"
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
	// A previous attempt's namespaces may still be terminating: deletion is
	// asynchronous, and applying into a terminating namespace fails.
	if err := r.waitGone(ctx); err != nil {
		return fmt.Errorf("the previous scenario has not finished tearing down: %w", err)
	}
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
	return r.Setup(ctx)
}

// waitGone blocks until every namespace this lab creates is absent. It
// returns immediately on a first run, when there is nothing to wait for.
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

// namespaceNames decodes every document because authoring supports both block
// and compact YAML metadata. Missing a namespace here creates a stop/start
// race: apply can reach the API server while the old namespace is terminating.
func namespaceNames(manifest string) []string {
	var out []string
	decoder := yaml.NewDecoder(strings.NewReader(manifest))
	for {
		var document struct {
			Kind     string `yaml:"kind"`
			Metadata struct {
				Name string `yaml:"name"`
			} `yaml:"metadata"`
		}
		err := decoder.Decode(&document)
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		if document.Kind == "Namespace" && document.Metadata.Name != "" {
			out = append(out, document.Metadata.Name)
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
	return gradeChecks(ctx, r.Env, r.Plan.Checks, r.Plan.AnyChecks)
}

// GradeCheckpoint evaluates the state requirements for one interactive step.
func (r *Runner) GradeCheckpoint(ctx context.Context, index int) (*Report, error) {
	if index < 0 || index >= len(r.Plan.Checkpoints) {
		return nil, fmt.Errorf("checkpoint %d is out of range", index+1)
	}
	cp := r.Plan.Checkpoints[index]
	return gradeChecks(ctx, r.Env, cp.Checks, cp.AnyChecks)
}

func gradeChecks(ctx context.Context, env *environment.Manager, checks, anyChecks []grader.Checker) (*Report, error) {
	rep := &Report{Passed: true}
	for _, c := range checks {
		res := c.Check(ctx, env)
		rep.Required = append(rep.Required, res)
		if !res.Passed {
			rep.Passed = false
		}
	}
	if len(anyChecks) > 0 {
		anyPassed := false
		for _, c := range anyChecks {
			res := c.Check(ctx, env)
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
