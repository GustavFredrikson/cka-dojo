// Package provider abstracts the machines a dojo environment runs on.
//
// The interface is deliberately node-granular: create/start/stop/destroy one
// machine, run a script on it, move a file in or out. Environment-level
// orchestration (which node is the control plane, in what order to provision)
// lives in internal/environment, so that adding a second provider never
// touches curriculum or lab code.
package provider

import (
	"context"
	"fmt"
	"io/fs"
)

// Status is the lifecycle state of a single node.
type Status string

const (
	StatusMissing Status = "missing"
	StatusStopped Status = "stopped"
	StatusRunning Status = "running"
	StatusBroken  Status = "broken"
	StatusUnknown Status = "unknown"
)

// NodeSpec describes a machine to create.
type NodeSpec struct {
	// Name is the provider-level instance name, e.g. cka-dojo-standard-cp1.
	Name string
	// Hostname is what the guest should call itself, e.g. cp1. Kubernetes
	// node names come from this, so it must be set before kubeadm runs.
	Hostname string
	CPUs     int
	Memory   string
	Disk     string
	// Network is the provider network joining the nodes of one environment.
	Network string
}

// NodeInfo is a discovered instance.
type NodeInfo struct {
	Name   string
	Status Status
	Dir    string
}

// ExecOptions describes one script execution inside a guest.
type ExecOptions struct {
	// Script is a shell script. It is fed to the guest on stdin, so it may
	// contain any characters without quoting concerns.
	Script string
	// User is "" (the provider's default user), "root", or a named user.
	User string
}

// ExecResult is the outcome of a script execution.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// ExitError is returned by Exec when the script exits non-zero.
type ExitError struct {
	Node   string
	Result ExecResult
}

func (e *ExitError) Error() string {
	msg := e.Result.Stderr
	if msg == "" {
		msg = e.Result.Stdout
	}
	return fmt.Sprintf("%s: command exited %d: %s", e.Node, e.Result.ExitCode, trim(msg))
}

func trim(s string) string {
	const max = 600
	if len(s) > max {
		return "..." + s[len(s)-max:]
	}
	return s
}

// Provider manages the machines an environment is made of.
type Provider interface {
	// Name identifies the implementation, e.g. "lima".
	Name() string
	// Available reports whether the provider can be used on this host.
	Available() error
	// EnsureNode creates the node if missing and starts it if stopped.
	EnsureNode(ctx context.Context, spec NodeSpec) error
	// StartNode starts an existing, stopped node.
	StartNode(ctx context.Context, name string) error
	// StopNode shuts a node down without deleting its disk.
	StopNode(ctx context.Context, name string) error
	// DestroyNode deletes a node and its disk.
	DestroyNode(ctx context.Context, name string) error
	// Status reports one node's lifecycle state.
	Status(ctx context.Context, name string) (Status, error)
	// List returns every node whose name starts with prefix.
	List(ctx context.Context, prefix string) ([]NodeInfo, error)
	// Exec runs a script in a guest and captures its output.
	Exec(ctx context.Context, name string, opts ExecOptions) (ExecResult, error)
	// WriteFile places content at path inside the guest, as root.
	WriteFile(ctx context.Context, name, path string, content []byte, mode fs.FileMode) error
	// ReadFile reads a file out of the guest, as root.
	ReadFile(ctx context.Context, name, path string) ([]byte, error)
	// Shell attaches an interactive shell, inheriting the caller's stdio.
	Shell(ctx context.Context, name, user string, args []string) error
}

// Run is a convenience wrapper: run a script as root and return stdout.
func Run(ctx context.Context, p Provider, node, script string) (string, error) {
	res, err := p.Exec(ctx, node, ExecOptions{Script: script, User: "root"})
	if err != nil {
		return res.Stdout, err
	}
	return res.Stdout, nil
}
