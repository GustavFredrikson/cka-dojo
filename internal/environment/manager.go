package environment

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/gustavfredrikson/cka-dojo/internal/config"
	"github.com/gustavfredrikson/cka-dojo/internal/content"
	"github.com/gustavfredrikson/cka-dojo/internal/provider"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
)

// NamePrefix namespaces every machine dojo creates, so `dojo env list` and
// `limactl list` stay legible next to a user's own VMs.
const NamePrefix = "cka-dojo-"

// StudentUser is the unprivileged account the learner works as. It exists so
// the shell prompt and SSH between nodes look like the real exam workstation.
const StudentUser = "student"

// AdminKubeconfig is the cluster-admin kubeconfig the engine grades through.
// The engine never uses the learner's kubeconfig: breaking it is a valid lab.
const AdminKubeconfig = "/etc/kubernetes/admin.conf"

// Manager drives one environment profile.
type Manager struct {
	Profile *Profile
	Prov    provider.Provider
	Src     *content.Source

	mu  sync.Mutex
	ips map[string]string
}

// New builds a Manager.
func New(p *Profile, prov provider.Provider, src *content.Source) *Manager {
	return &Manager{Profile: p, Prov: prov, Src: src, ips: map[string]string{}}
}

// VMName is the provider-level name of a node in this profile.
func (m *Manager) VMName(node string) string {
	return NamePrefix + m.Profile.ID + "-" + node
}

// Prefix is the provider-level name prefix for this profile.
func (m *Manager) Prefix() string { return NamePrefix + m.Profile.ID + "-" }

// NodeStatus pairs a node with its lifecycle state.
type NodeStatus struct {
	Node   NodeConfig
	VMName string
	Status provider.Status
	IP     string
}

// Status reports every node's state. It never fails just because a node is
// missing: dojo env status has to work on a half-built environment.
func (m *Manager) Status(ctx context.Context) ([]NodeStatus, error) {
	if err := m.Profile.validateIDs(); err != nil {
		return nil, err
	}
	out := make([]NodeStatus, 0, len(m.Profile.Nodes))
	for _, n := range m.Profile.Nodes {
		vm := m.VMName(n.Name)
		st, err := m.Prov.Status(ctx, vm)
		if err != nil {
			return nil, err
		}
		ns := NodeStatus{Node: n, VMName: vm, Status: st}
		if st == provider.StatusRunning {
			if ip, err := m.nodeIP(ctx, n.Name); err == nil {
				ns.IP = ip
			}
		}
		out = append(out, ns)
	}
	return out, nil
}

// Running reports whether every node in the profile is up.
func (m *Manager) Running(ctx context.Context) (bool, error) {
	sts, err := m.Status(ctx)
	if err != nil {
		return false, err
	}
	for _, s := range sts {
		if s.Status != provider.StatusRunning {
			return false, nil
		}
	}
	return len(sts) > 0, nil
}

// Exec runs a script on a node of this profile.
func (m *Manager) Exec(ctx context.Context, node, script, user string) (provider.ExecResult, error) {
	if err := m.Profile.validateIDs(); err != nil {
		return provider.ExecResult{}, err
	}
	if err := config.ValidateID(node); err != nil {
		return provider.ExecResult{}, fmt.Errorf("node name: %w", err)
	}
	return m.Prov.Exec(ctx, m.VMName(node), provider.ExecOptions{Script: script, User: user})
}

// Run executes a script as root and returns trimmed stdout.
func (m *Manager) Run(ctx context.Context, node, script string) (string, error) {
	res, err := m.Exec(ctx, node, script, "root")
	return strings.TrimSpace(res.Stdout), err
}

// EnsureNodes creates and starts every machine, in parallel. Boot dominates
// setup time, so serial creation would roughly quadruple it.
func (m *Manager) EnsureNodes(ctx context.Context) error {
	if err := m.Profile.validateIDs(); err != nil {
		return err
	}
	var wg sync.WaitGroup
	errs := make([]error, len(m.Profile.Nodes))
	for i, n := range m.Profile.Nodes {
		wg.Add(1)
		go func(i int, n NodeConfig) {
			defer wg.Done()
			ui.Step("preparing machine %s (%d vCPU, %s)", n.Name, n.CPUs, n.Memory)
			errs[i] = m.Prov.EnsureNode(ctx, provider.NodeSpec{
				Name:     m.VMName(n.Name),
				Hostname: n.Name,
				CPUs:     n.CPUs,
				Memory:   n.Memory,
				Disk:     n.Disk,
				Network:  m.Profile.Network.Lima,
			})
			if errs[i] == nil {
				ui.OK("machine %s is up", n.Name)
			}
		}(i, n)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			return fmt.Errorf("node %s: %w", m.Profile.Nodes[i].Name, err)
		}
	}
	return nil
}

// Stop shuts every node down, keeping disks so switching profiles is cheap.
func (m *Manager) Stop(ctx context.Context) error {
	if err := m.Profile.validateIDs(); err != nil {
		return err
	}
	for _, n := range m.Profile.Nodes {
		ui.Step("stopping %s", n.Name)
		if err := m.Prov.StopNode(ctx, m.VMName(n.Name)); err != nil {
			return err
		}
	}
	return nil
}

// StopOthers shuts down every dojo machine belonging to a different profile.
//
// Only one environment is meant to run at a time. Nothing enforced that until
// now, and two at once is a memory budget nobody has: the standard profile
// alone wants about 9 GiB of guest RAM. The host starts swapping long before
// the second cluster finishes booting, and the first symptom is etcd losing
// quorum -- which reads as a bug in whichever profile was started second.
//
// Disks are kept, so switching back is a start rather than a rebuild.
func (m *Manager) StopOthers(ctx context.Context) error {
	if err := m.Profile.validateIDs(); err != nil {
		return err
	}
	nodes, err := m.Prov.List(ctx, NamePrefix)
	if err != nil {
		return err
	}
	for _, n := range nodes {
		if strings.HasPrefix(n.Name, m.Prefix()) || n.Status != provider.StatusRunning {
			continue
		}
		ui.Step("stopping %s (it belongs to another profile)", n.Name)
		if err := m.Prov.StopNode(ctx, n.Name); err != nil {
			return fmt.Errorf("stop %s: %w", n.Name, err)
		}
	}
	return nil
}

// Destroy deletes every node of this profile.
func (m *Manager) Destroy(ctx context.Context) error {
	if err := m.Profile.validateIDs(); err != nil {
		return err
	}
	if _, err := config.EnvDirPath(m.Profile.ID); err != nil {
		return fmt.Errorf("environment artifacts: %w", err)
	}
	for i := len(m.Profile.Nodes) - 1; i >= 0; i-- {
		n := m.Profile.Nodes[i]
		ui.Step("destroying %s", n.Name)
		if err := m.Prov.DestroyNode(ctx, m.VMName(n.Name)); err != nil {
			return err
		}
	}
	if err := config.RemoveEnvDir(m.Profile.ID); err != nil {
		return fmt.Errorf("remove environment artifacts: %w", err)
	}
	m.mu.Lock()
	m.ips = map[string]string{}
	m.mu.Unlock()
	return nil
}

// nodeIP returns a node's address on the environment network.
//
// The address is discovered by matching the profile's subnet rather than by
// taking "the primary interface", because the interface layout depends on how
// the provider is configured. Lima's default user-mode network gives every VM
// the same address (192.168.5.15); the user-v2 network replaces that NIC and
// hands out distinct ones. Matching the subnet is correct either way.
func (m *Manager) nodeIP(ctx context.Context, node string) (string, error) {
	m.mu.Lock()
	if ip, ok := m.ips[node]; ok {
		m.mu.Unlock()
		return ip, nil
	}
	m.mu.Unlock()

	prefix := subnetPrefix(m.Profile.Network.Subnet)
	script := fmt.Sprintf(`set -euo pipefail
ip -4 -o addr show scope global | awk '{print $4}' | cut -d/ -f1 | grep '^%s' | head -1
`, prefix)
	out, err := m.Run(ctx, node, script)
	if err != nil {
		return "", fmt.Errorf("discover %s address on %s: %w", m.Profile.Network.Subnet, node, err)
	}
	if out == "" {
		return "", fmt.Errorf("node %s has no address on %s; is the lima %q network up?",
			node, m.Profile.Network.Subnet, m.Profile.Network.Lima)
	}
	m.mu.Lock()
	m.ips[node] = out
	m.mu.Unlock()
	return out, nil
}

// NodeIP is the exported form of nodeIP.
func (m *Manager) NodeIP(ctx context.Context, node string) (string, error) {
	return m.nodeIP(ctx, node)
}

// subnetPrefix turns 192.168.104.0/24 into "192.168.104." for matching.
func subnetPrefix(cidr string) string {
	host := cidr
	if i := strings.Index(cidr, "/"); i > 0 {
		host = cidr[:i]
	}
	parts := strings.Split(host, ".")
	if len(parts) != 4 {
		return host
	}
	return strings.Join(parts[:3], ".") + "."
}

// allIPs discovers every node's address.
func (m *Manager) allIPs(ctx context.Context) (map[string]string, error) {
	out := map[string]string{}
	for _, n := range m.Profile.Nodes {
		ip, err := m.nodeIP(ctx, n.Name)
		if err != nil {
			return nil, err
		}
		out[n.Name] = ip
	}
	return out, nil
}
