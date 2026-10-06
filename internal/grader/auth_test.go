package grader

import (
	"context"
	"errors"
	"testing"

	"github.com/gustavfredrikson/cka-dojo/internal/provider"
)

func TestAuthCanIRequiresAnExplicitPermissionAnswer(t *testing.T) {
	for _, tc := range []struct {
		name       string
		res        provider.ExecResult
		err        error
		expect     bool
		wantPassed bool
		wantError  bool
	}{
		{name: "permission allowed", res: provider.ExecResult{Stdout: "yes\n"}, expect: true, wantPassed: true},
		{name: "permission denied", res: provider.ExecResult{Stdout: "no\n", ExitCode: 1}, wantPassed: true},
		{name: "surrounding whitespace", res: provider.ExecResult{Stdout: " \tno\n", ExitCode: 1}, wantPassed: true},
		{name: "allowed when denial required", res: provider.ExecResult{Stdout: "yes\n"}},
		{name: "denied when allowance required", res: provider.ExecResult{Stdout: "no\n", ExitCode: 1}, expect: true},
		{name: "API unavailable", res: provider.ExecResult{Stderr: "connection refused", ExitCode: 1}, wantError: true},
		{name: "empty success", wantError: true},
		{name: "empty denial", res: provider.ExecResult{ExitCode: 1}, wantError: true},
		{name: "garbage", res: provider.ExecResult{Stdout: "unknown\n", ExitCode: 1}, wantError: true},
		{name: "yes prefix is not yes", res: provider.ExecResult{Stdout: "yes but unavailable\n"}, expect: true, wantError: true},
		{name: "multiple answers", res: provider.ExecResult{Stdout: "no\nyes\n", ExitCode: 1}, wantError: true},
		{name: "yes with failure exit", res: provider.ExecResult{Stdout: "yes\n", ExitCode: 1}, expect: true, wantError: true},
		{name: "no with success exit", res: provider.ExecResult{Stdout: "no\n"}, wantError: true},
		{name: "no with transport exit", res: provider.ExecResult{Stdout: "no\n", ExitCode: 255}, wantError: true},
		{name: "transport error", err: errors.New("SSH transport unavailable"), wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.err
			if err == nil && tc.res.ExitCode != 0 {
				err = &provider.ExitError{Node: "cp1", Result: tc.res}
			}
			check := &AuthCanI{User: "alice", Namespace: "backend", Verb: "get", Resource: "pods", Expect: &tc.expect}
			result := check.Check(context.Background(), graderEnvironment(tc.res, err))
			if result.Passed != tc.wantPassed || (result.Err != nil) != tc.wantError {
				t.Fatalf("result = %+v, want passed=%v, error=%v", result, tc.wantPassed, tc.wantError)
			}
			if !tc.wantPassed && !tc.wantError && result.Detail == "" {
				t.Fatal("a valid unexpected permission needs an explanation")
			}
		})
	}
}
