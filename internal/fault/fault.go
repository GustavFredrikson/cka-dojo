// Package fault is the vocabulary a lab author uses to break a cluster.
//
// Faults are deliberately small and composable: a handful of primitives
// generate dozens of troubleshooting scenarios, and every one of them can be
// undone well enough for `dojo reset` to put the lab back.
package fault

import (
	"context"
	"fmt"
	"io/fs"
	"sort"

	"github.com/gustavfredrikson/cka-dojo/internal/environment"
	"github.com/gustavfredrikson/cka-dojo/internal/spec"
)

// Context is what a fault is allowed to reach: the lab's own files, and
// nothing else.
type Context struct {
	// Files is the lab directory, for manifests and replacement file bodies.
	Files fs.FS
	// LabID identifies the lab, for labelling and log lines.
	LabID string
}

// Fault is one thing done to an environment to create a scenario.
type Fault interface {
	// Type is the discriminator used in lab.yaml.
	Type() string
	// Validate checks the decoded fields without touching a cluster, so that
	// `dojo content validate` can catch mistakes offline.
	Validate() error
	// Inject breaks the environment.
	Inject(ctx context.Context, env *environment.Manager, lc *Context) error
	// Repair undoes the fault on a best-effort basis. Anything a reset can
	// achieve by re-applying the baseline may be a no-op here.
	Repair(ctx context.Context, env *environment.Manager, lc *Context) error
	// Describe is a one-line summary for verbose output and logs.
	Describe() string
}

// factory builds an empty fault of one type.
type factory func() Fault

var registry = map[string]factory{}

func register(f factory) {
	registry[f().Type()] = f
}

// Types lists every registered fault type.
func Types() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Build turns a lab.yaml entry into a validated fault.
func Build(s spec.Spec) (Fault, error) {
	mk, ok := registry[s.Type]
	if !ok {
		return nil, fmt.Errorf("unknown fault type %q (known: %v)", s.Type, Types())
	}
	f := mk()
	if err := s.Decode(f); err != nil {
		return nil, err
	}
	if err := f.Validate(); err != nil {
		return nil, fmt.Errorf("fault %s (line %d): %w", s.Type, s.Line(), err)
	}
	return f, nil
}

// BuildAll builds a list of faults, reporting the first problem.
func BuildAll(specs []spec.Spec) ([]Fault, error) {
	out := make([]Fault, 0, len(specs))
	for _, s := range specs {
		f, err := Build(s)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}
