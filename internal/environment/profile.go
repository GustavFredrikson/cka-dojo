// Package environment turns an environment profile into running machines.
package environment

import (
	"fmt"
	"path"
	"strings"

	"github.com/gustavfredrikson/cka-dojo/internal/content"
	"gopkg.in/yaml.v3"
)

// Node roles. The role drives which provisioning steps a node gets, which is
// what lets a second profile reuse the same engine.
const (
	RoleWorkstation  = "workstation"
	RoleControlPlane = "control-plane"
	RoleWorker       = "worker"
	// RoleBlank is a prepared Linux box with no Kubernetes on it, for the
	// future `raw` profile where installing Kubernetes is the exercise.
	RoleBlank = "blank"
)

// Provisioning modes.
const (
	ProvisionKubeadm = "kubeadm"
	ProvisionNone    = "none"
)

// Profile is environments/<id>/environment.yaml.
type Profile struct {
	SchemaVersion int    `yaml:"schemaVersion"`
	ID            string `yaml:"id"`
	Description   string `yaml:"description"`
	// Provisioning selects how far the engine takes the machines.
	Provisioning string `yaml:"provisioning"`

	Kubernetes struct {
		// Version is the exact patch, e.g. 1.35.8. Never "latest": the exam
		// tracks a minor that lags upstream.
		Version       string `yaml:"version"`
		PodSubnet     string `yaml:"podSubnet"`
		ServiceSubnet string `yaml:"serviceSubnet"`
	} `yaml:"kubernetes"`

	CNI struct {
		Provider string `yaml:"provider"`
		Version  string `yaml:"version"`
	} `yaml:"cni"`

	Addons struct {
		MetricsServer string `yaml:"metricsServer"`
		Helm          string `yaml:"helm"`
	} `yaml:"addons"`

	Network struct {
		// Lima is the provider network name joining the nodes.
		Lima string `yaml:"lima"`
		// Subnet is that network's CIDR. Node IPs are discovered by matching
		// it, because Lima's default route interface has the same address on
		// every VM.
		Subnet string `yaml:"subnet"`
	} `yaml:"network"`

	Nodes []NodeConfig `yaml:"nodes"`
}

// NodeConfig is one machine in a profile.
type NodeConfig struct {
	Name   string `yaml:"name"`
	Role   string `yaml:"role"`
	CPUs   int    `yaml:"cpus"`
	Memory string `yaml:"memory"`
	Disk   string `yaml:"disk"`
}

// Minor returns the Kubernetes minor version, e.g. "1.35".
func (p *Profile) Minor() string {
	parts := strings.Split(p.Kubernetes.Version, ".")
	if len(parts) < 2 {
		return p.Kubernetes.Version
	}
	return parts[0] + "." + parts[1]
}

// Dir is the content directory holding this profile.
func (p *Profile) Dir() string { return path.Join("environments", p.ID) }

// NodeByName finds a node, or nil.
func (p *Profile) NodeByName(name string) *NodeConfig {
	for i := range p.Nodes {
		if p.Nodes[i].Name == name {
			return &p.Nodes[i]
		}
	}
	return nil
}

// NodesByRole returns every node with the given role, in declaration order.
func (p *Profile) NodesByRole(role string) []*NodeConfig {
	var out []*NodeConfig
	for i := range p.Nodes {
		if p.Nodes[i].Role == role {
			out = append(out, &p.Nodes[i])
		}
	}
	return out
}

// ControlPlane returns the first control-plane node, or nil.
func (p *Profile) ControlPlane() *NodeConfig {
	if n := p.NodesByRole(RoleControlPlane); len(n) > 0 {
		return n[0]
	}
	return nil
}

// Workstation returns the terminal node, or nil.
func (p *Profile) Workstation() *NodeConfig {
	if n := p.NodesByRole(RoleWorkstation); len(n) > 0 {
		return n[0]
	}
	return nil
}

// Validate checks a profile for the mistakes that would otherwise surface as
// a confusing failure twenty minutes into provisioning.
func (p *Profile) Validate() error {
	if p.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schemaVersion %d (want 1)", p.SchemaVersion)
	}
	if p.ID == "" {
		return fmt.Errorf("id is required")
	}
	if len(p.Nodes) == 0 {
		return fmt.Errorf("profile has no nodes")
	}
	switch p.Provisioning {
	case ProvisionKubeadm, ProvisionNone:
	case "":
		p.Provisioning = ProvisionKubeadm
	default:
		return fmt.Errorf("unknown provisioning mode %q", p.Provisioning)
	}
	seen := map[string]bool{}
	for _, n := range p.Nodes {
		if n.Name == "" {
			return fmt.Errorf("node without a name")
		}
		if seen[n.Name] {
			return fmt.Errorf("duplicate node %q", n.Name)
		}
		seen[n.Name] = true
		switch n.Role {
		case RoleWorkstation, RoleControlPlane, RoleWorker, RoleBlank:
		default:
			return fmt.Errorf("node %s: unknown role %q", n.Name, n.Role)
		}
		if n.CPUs <= 0 || n.Memory == "" || n.Disk == "" {
			return fmt.Errorf("node %s: cpus, memory and disk are required", n.Name)
		}
	}
	if p.Provisioning == ProvisionKubeadm {
		if p.Kubernetes.Version == "" {
			return fmt.Errorf("kubernetes.version is required (pin an exact patch, never `latest`)")
		}
		if strings.Contains(p.Kubernetes.Version, "latest") {
			return fmt.Errorf("kubernetes.version must be an exact patch, not %q", p.Kubernetes.Version)
		}
		if p.ControlPlane() == nil {
			return fmt.Errorf("kubeadm profile needs a control-plane node")
		}
		if p.Kubernetes.PodSubnet == "" || p.Kubernetes.ServiceSubnet == "" {
			return fmt.Errorf("kubernetes.podSubnet and serviceSubnet are required")
		}
		if strings.HasPrefix(p.Kubernetes.PodSubnet, trimCIDR(p.Network.Subnet)) {
			return fmt.Errorf("podSubnet %s overlaps the node network %s", p.Kubernetes.PodSubnet, p.Network.Subnet)
		}
	}
	if p.Network.Lima != "" && p.Network.Subnet == "" {
		return fmt.Errorf("network.subnet is required when network.lima is set")
	}
	return nil
}

func trimCIDR(cidr string) string {
	if i := strings.Index(cidr, "/"); i > 0 {
		return cidr[:i]
	}
	return cidr
}

// LoadProfile reads environments/<id>/environment.yaml.
func LoadProfile(src *content.Source, id string) (*Profile, error) {
	name := path.Join("environments", id, "environment.yaml")
	data, err := src.Read(name)
	if err != nil {
		return nil, fmt.Errorf("environment profile %q not found (%s)", id, name)
	}
	p := &Profile{}
	if err := yaml.Unmarshal(data, p); err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	if p.ID == "" {
		p.ID = id
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return p, nil
}

// ListProfiles returns every profile id found in the content root.
func ListProfiles(src *content.Source) ([]string, error) {
	entries, err := fsReadDir(src, "environments")
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() && src.Exists(path.Join("environments", e.Name(), "environment.yaml")) {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}
