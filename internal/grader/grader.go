// Package grader evaluates whether the cluster is now in the state a lab
// asked for.
//
// Graders never look at what the learner typed. Imperative kubectl, an edited
// YAML file, `kubectl patch` and `kubectl edit` are all correct if the
// resulting state is correct, which is exactly how the exam scores.
package grader

import (
	"context"
	"fmt"
	"sort"

	"github.com/gustavfredrikson/cka-dojo/internal/environment"
	"github.com/gustavfredrikson/cka-dojo/internal/spec"
)

// Result is one check's outcome.
type Result struct {
	// Description is what was being checked, phrased as a requirement.
	Description string
	Passed      bool
	// Detail explains a failure without revealing the fix.
	Detail string
	// Err is set when the check itself could not run.
	Err error
}

// Checker is one graded requirement.
type Checker interface {
	// Type is the discriminator used in lab.yaml.
	Type() string
	// Validate checks the decoded fields offline.
	Validate() error
	// Describe states the requirement.
	Describe() string
	// Check evaluates the live cluster.
	Check(ctx context.Context, env *environment.Manager) Result
}

type factory func() Checker

var registry = map[string]factory{}

func register(f factory) { registry[f().Type()] = f }

// Types lists every registered grader type.
func Types() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Build turns a lab.yaml entry into a validated checker.
func Build(s spec.Spec) (Checker, error) {
	mk, ok := registry[s.Type]
	if !ok {
		return nil, fmt.Errorf("unknown grader type %q (known: %v)", s.Type, Types())
	}
	c := mk()
	if err := s.Decode(c); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("grader %s (line %d): %w", s.Type, s.Line(), err)
	}
	return c, nil
}

// BuildAll builds a list of checkers.
func BuildAll(specs []spec.Spec) ([]Checker, error) {
	out := make([]Checker, 0, len(specs))
	for _, s := range specs {
		c, err := Build(s)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// pass and fail keep the checkers below terse.
func pass(desc string) Result { return Result{Description: desc, Passed: true} }

func fail(desc, detail string, a ...any) Result {
	return Result{Description: desc, Detail: fmt.Sprintf(detail, a...)}
}

func broken(desc string, err error) Result {
	return Result{Description: desc, Err: err}
}
