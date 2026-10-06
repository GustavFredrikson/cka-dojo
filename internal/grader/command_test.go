package grader

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/environment"
	"github.com/gustavfredrikson/cka-dojo/internal/provider"
	"github.com/gustavfredrikson/cka-dojo/internal/provider/fake"
)

func graderEnvironment(res provider.ExecResult, err error) *environment.Manager {
	p := fake.New()
	p.Responder = func(node, script string) (provider.ExecResult, error) {
		return res, err
	}
	return environment.New(&environment.Profile{
		ID:    "test",
		Nodes: []environment.NodeConfig{{Name: "cp1", Role: environment.RoleControlPlane}},
	}, p, nil)
}

func TestCommandRejectsExecutionErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"provider transport", errors.New("SSH transport unavailable")},
		{"executable launch", &exec.Error{Name: "limactl", Err: exec.ErrNotFound}},
		{"canceled", context.Canceled},
		{"deadline", fmt.Errorf("execute: %w", context.DeadlineExceeded)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			check := &Command{Command: "true"}
			result := check.Check(context.Background(), graderEnvironment(provider.ExecResult{}, tc.err))
			if result.Passed || !errors.Is(result.Err, tc.err) {
				t.Fatalf("result = %+v, want an unavailable check preserving %v", result, tc.err)
			}
		})
	}
}

func TestCommandEvaluatesRealCommandExits(t *testing.T) {
	one, two := 1, 2
	for _, tc := range []struct {
		name       string
		res        provider.ExecResult
		err        error
		wantExit   *int
		stdout     *Match
		wantPassed bool
	}{
		{name: "success", wantPassed: true},
		{
			name: "unexpected nonzero exit", res: provider.ExecResult{ExitCode: 1},
			err: &provider.ExitError{Node: "cp1", Result: provider.ExecResult{ExitCode: 1}},
		},
		{
			name: "expected nonzero exit", res: provider.ExecResult{ExitCode: 1},
			err:      &provider.ExitError{Node: "cp1", Result: provider.ExecResult{ExitCode: 1}},
			wantExit: &one, wantPassed: true,
		},
		{
			name: "wrapped expected exit", res: provider.ExecResult{ExitCode: 1},
			err:      fmt.Errorf("execute: %w", &provider.ExitError{Node: "cp1", Result: provider.ExecResult{ExitCode: 1}}),
			wantExit: &one, wantPassed: true,
		},
		{
			name: "wrong expected exit", res: provider.ExecResult{ExitCode: 1},
			err:      &provider.ExitError{Node: "cp1", Result: provider.ExecResult{ExitCode: 1}},
			wantExit: &two,
		},
		{
			name: "matching output", res: provider.ExecResult{Stdout: "healthy\n"},
			stdout: &Match{Contains: "healthy"}, wantPassed: true,
		},
		{
			name: "wrong output", res: provider.ExecResult{Stdout: "unavailable\n"},
			stdout: &Match{Contains: "healthy"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			check := &Command{Command: "check", ExitCode: tc.wantExit, Stdout: tc.stdout}
			result := check.Check(context.Background(), graderEnvironment(tc.res, tc.err))
			if result.Passed != tc.wantPassed || result.Err != nil {
				t.Fatalf("result = %+v, want passed=%v with no execution error", result, tc.wantPassed)
			}
			if !tc.wantPassed && result.Detail == "" {
				t.Fatal("a failed command check needs an explanation")
			}
		})
	}
}

func TestCommandRejectsCanceledExecutionContext(t *testing.T) {
	one := 1
	for _, tc := range []struct {
		name string
		ctx  func() (context.Context, context.CancelFunc)
		want error
	}{
		{"canceled", func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, cancel
		}, context.Canceled},
		{"expired deadline", func() (context.Context, context.CancelFunc) {
			return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		}, context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.ctx()
			defer cancel()
			res := provider.ExecResult{ExitCode: 1}
			err := &provider.ExitError{Node: "cp1", Result: res}
			check := &Command{Command: "check", ExitCode: &one}
			result := check.Check(ctx, graderEnvironment(res, err))
			if result.Passed || !errors.Is(result.Err, tc.want) {
				t.Fatalf("result = %+v, want unavailable check with %v", result, tc.want)
			}
		})
	}
}
