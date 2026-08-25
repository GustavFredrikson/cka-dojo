// Package fake is an in-memory provider for tests. It lets the engine's
// orchestration be exercised without booting a virtual machine, which is what
// keeps the fast test suite fast.
package fake

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/gustavfredrikson/cka-dojo/internal/provider"
)

// Call records one script execution.
type Call struct {
	Node   string
	User   string
	Script string
}

// Provider implements provider.Provider against maps.
type Provider struct {
	mu sync.Mutex

	// Nodes maps instance name to status.
	Nodes map[string]provider.Status
	// Files maps "node:path" to content.
	Files map[string][]byte
	// Calls records every Exec, in order.
	Calls []Call
	// Responder returns canned output for a script. The default responder
	// answers address-discovery queries and succeeds at everything else.
	Responder func(node, script string) (provider.ExecResult, error)
	// FailEnsure makes EnsureNode fail for a named node.
	FailEnsure map[string]error
}

// New returns a fake provider with no nodes.
func New() *Provider {
	return &Provider{
		Nodes: map[string]provider.Status{},
		Files: map[string][]byte{},
	}
}

// Name implements provider.Provider.
func (p *Provider) Name() string { return "fake" }

// Available implements provider.Provider.
func (p *Provider) Available() error { return nil }

// EnsureNode implements provider.Provider.
func (p *Provider) EnsureNode(ctx context.Context, spec provider.NodeSpec) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err, ok := p.FailEnsure[spec.Name]; ok {
		return err
	}
	p.Nodes[spec.Name] = provider.StatusRunning
	return nil
}

// StartNode implements provider.Provider.
func (p *Provider) StartNode(ctx context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Nodes[name] = provider.StatusRunning
	return nil
}

// StopNode implements provider.Provider.
func (p *Provider) StopNode(ctx context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Nodes[name] = provider.StatusStopped
	return nil
}

// DestroyNode implements provider.Provider.
func (p *Provider) DestroyNode(ctx context.Context, name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.Nodes, name)
	return nil
}

// Status implements provider.Provider.
func (p *Provider) Status(ctx context.Context, name string) (provider.Status, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if st, ok := p.Nodes[name]; ok {
		return st, nil
	}
	return provider.StatusMissing, nil
}

// List implements provider.Provider.
func (p *Provider) List(ctx context.Context, prefix string) ([]provider.NodeInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []provider.NodeInfo
	for name, st := range p.Nodes {
		if strings.HasPrefix(name, prefix) {
			out = append(out, provider.NodeInfo{Name: name, Status: st})
		}
	}
	return out, nil
}

// Exec implements provider.Provider.
func (p *Provider) Exec(ctx context.Context, name string, opts provider.ExecOptions) (provider.ExecResult, error) {
	p.mu.Lock()
	p.Calls = append(p.Calls, Call{Node: name, User: opts.User, Script: opts.Script})
	responder := p.Responder
	p.mu.Unlock()
	if responder != nil {
		return responder(name, opts.Script)
	}
	return defaultResponder(name, opts.Script)
}

// defaultResponder answers the queries the environment manager makes during a
// normal bring-up.
func defaultResponder(name, script string) (provider.ExecResult, error) {
	switch {
	case strings.Contains(script, "ip -4 -o addr show"):
		// Hand out a stable address per node so tests can assert on
		// /etc/hosts content.
		return provider.ExecResult{Stdout: "192.168.104." + fmt.Sprint(10+len(name)%40) + "\n"}, nil
	case strings.Contains(script, "test -f"):
		return provider.ExecResult{Stdout: "no\n"}, nil
	default:
		return provider.ExecResult{}, nil
	}
}

// WriteFile implements provider.Provider.
func (p *Provider) WriteFile(ctx context.Context, name, path string, content []byte, mode fs.FileMode) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Files[name+":"+path] = content
	return nil
}

// ReadFile implements provider.Provider.
func (p *Provider) ReadFile(ctx context.Context, name, path string) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if b, ok := p.Files[name+":"+path]; ok {
		return b, nil
	}
	return nil, fmt.Errorf("fake: no file %s on %s", path, name)
}

// Shell implements provider.Provider.
func (p *Provider) Shell(ctx context.Context, name, user string, args []string) error { return nil }

// ScriptsFor returns every script run on one node.
func (p *Provider) ScriptsFor(node string) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []string
	for _, c := range p.Calls {
		if c.Node == node {
			out = append(out, c.Script)
		}
	}
	return out
}

// Ran reports whether any script on any node contained the substring.
func (p *Provider) Ran(substr string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.Calls {
		if strings.Contains(c.Script, substr) {
			return true
		}
	}
	return false
}

var _ provider.Provider = (*Provider)(nil)
