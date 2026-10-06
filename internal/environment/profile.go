// Package environment turns an environment profile into running machines.
package environment

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/gustavfredrikson/cka-dojo/internal/config"
	"github.com/gustavfredrikson/cka-dojo/internal/content"
	"gopkg.in/yaml.v3"
)

// Node roles. The role drives which provisioning steps a node gets, which is
// what lets a second profile reuse the same engine.
const (
	RoleWorkstation  = "workstation"
	RoleControlPlane = "control-plane"
	RoleWorker       = "worker"
)

// Provisioning modes.
const (
	ProvisionKubeadm = "kubeadm"
	// ProvisionNodes prepares every cluster node as far as kube-node.sh takes
	// it -- containerd, the kubeadm toolchain at the pinned patch, swap off,
	// the sysctls, the --node-ip pin -- and stops. Nothing is initialised:
	// installing Kubernetes is the exercise. Used by the `raw` profile.
	//
	// Those nodes still declare roles control-plane and worker, because a role
	// states intent rather than current state. That is what makes
	// Profile.ControlPlane() return a node graders can run kubectl on once the
	// learner has built a cluster -- and before that, kubectl simply exits
	// non-zero, which graders report as a failed check rather than an error.
	ProvisionNodes = "nodes"
	ProvisionNone  = "none"
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
		// LocalPathProvisioner backs the dynamic-provisioning exercises. A
		// kubeadm cluster on plain VMs has no CSI driver, so without it
		// "implement storage classes and dynamic volume provisioning" cannot
		// be practised at all.
		LocalPathProvisioner string `yaml:"localPathProvisioner"`
		// IngressNginx is the Ingress controller. Curriculum revision
		// 2025-02-18 examines Ingress resources and controllers directly.
		IngressNginx string `yaml:"ingressNginx"`
		// GatewayAPI is the CRD bundle; NginxGatewayFabric is the controller
		// that makes those objects do something. Both are needed for a
		// Gateway to reach Programmed rather than sitting Unaccepted.
		GatewayAPI         string `yaml:"gatewayAPI"`
		NginxGatewayFabric string `yaml:"nginxGatewayFabric"`
	} `yaml:"addons"`

	// APIEndpoint is the stable address clients and joining control planes use.
	// It describes a machine rather than a Kubernetes setting, which is why it
	// sits beside `kubernetes` rather than inside it.
	//
	// It must exist before `kubeadm init`: --control-plane-endpoint is written
	// into the API server certificate's SANs, into kube-proxy's ConfigMap and
	// into every kubeconfig, and cannot be added afterwards without reissuing
	// certificates.
	APIEndpoint struct {
		// Node runs the load balancer. Deliberately not a control plane: the
		// address behind this name is baked into certificates at init time, so
		// it belongs on the one machine labs are not allowed to break.
		Node string `yaml:"node"`
		// Name goes into the /etc/hosts block the engine already owns, so the
		// endpoint is a name rather than an address. If the load balancer's
		// address ever moves, the fix is one rewritten hosts line instead of
		// reissued certificates.
		Name string `yaml:"name"`
		Port int    `yaml:"port"`
	} `yaml:"apiEndpoint"`

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

// ClusterNodes returns the nodes that become part of the cluster: control
// planes first, then workers, each in declaration order. The workstation is
// not one of them -- it runs no kubelet, which is what lets a lab break the
// control plane without breaking the shell the learner works from.
func (p *Profile) ClusterNodes() []*NodeConfig {
	out := p.NodesByRole(RoleControlPlane)
	return append(out, p.NodesByRole(RoleWorker)...)
}

// ControlPlanes returns every control-plane node, in declaration order. The
// first is the one `kubeadm init` runs on; the rest join it.
func (p *Profile) ControlPlanes() []*NodeConfig {
	return p.NodesByRole(RoleControlPlane)
}

// ControlPlane returns the first control-plane node, or nil.
//
// Its contract is "a node the engine can run kubectl and stage manifests on",
// which is a correct answer on any profile with at least one control plane.
// Callers that mean "every control plane" want ControlPlanes.
func (p *Profile) ControlPlane() *NodeConfig {
	if n := p.NodesByRole(RoleControlPlane); len(n) > 0 {
		return n[0]
	}
	return nil
}

// HasAPIEndpoint reports whether this profile puts a load balancer in front of
// the API server.
func (p *Profile) HasAPIEndpoint() bool { return p.APIEndpoint.Name != "" }

// APIEndpointAddr is the host:port form kubeadm and kubeconfigs use, or "".
func (p *Profile) APIEndpointAddr() string {
	if !p.HasAPIEndpoint() {
		return ""
	}
	return fmt.Sprintf("%s:%d", p.APIEndpoint.Name, p.APIEndpoint.Port)
}

// Workstation returns the terminal node, or nil.
func (p *Profile) Workstation() *NodeConfig {
	if n := p.NodesByRole(RoleWorkstation); len(n) > 0 {
		return n[0]
	}
	return nil
}

// TotalMemoryGiB is the guest memory this profile asks for, across every node.
// `dojo doctor` sizes the host against it, so a heavier profile raises the bar
// on its own rather than needing the advice updated by hand.
func (p *Profile) TotalMemoryGiB() float64 {
	var total float64
	for _, n := range p.Nodes {
		total += sizeGiB(n.Memory)
	}
	return total
}

// TotalDiskGiB is the disk this profile asks for. Disks are thin-provisioned,
// so a built environment uses well under this -- it is an upper bound, which
// is the useful direction for a free-space check.
func (p *Profile) TotalDiskGiB() float64 {
	var total float64
	for _, n := range p.Nodes {
		total += sizeGiB(n.Disk)
	}
	return total
}

// sizeGiB parses the memory and disk spellings Lima accepts. An unparseable
// value returns 0 rather than an error: these feed advisory host checks, and
// a profile that got this wrong fails at Validate.
func sizeGiB(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	units := []struct {
		suffix string
		factor float64
	}{
		{"GiB", 1},
		{"MiB", 1.0 / 1024},
		{"KiB", 1.0 / (1024 * 1024)},
		{"G", 1},
		{"M", 1.0 / 1024},
		{"K", 1.0 / (1024 * 1024)},
	}
	for _, u := range units {
		if rest, ok := strings.CutSuffix(s, u.suffix); ok {
			n, err := strconv.ParseFloat(strings.TrimSpace(rest), 64)
			if err != nil {
				return 0
			}
			return n * u.factor
		}
	}
	// A bare number is bytes, as in Lima's own schema.
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return n / (1024 * 1024 * 1024)
}

// Validate checks a profile for the mistakes that would otherwise surface as
// a confusing failure twenty minutes into provisioning.
func (p *Profile) Validate() error {
	if p.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schemaVersion %d (want 1)", p.SchemaVersion)
	}
	if err := p.validateIDs(); err != nil {
		return err
	}
	if len(p.Nodes) == 0 {
		return fmt.Errorf("profile has no nodes")
	}
	switch p.Provisioning {
	case ProvisionKubeadm, ProvisionNodes, ProvisionNone:
	case "":
		p.Provisioning = ProvisionKubeadm
	default:
		return fmt.Errorf("unknown provisioning mode %q", p.Provisioning)
	}
	seen := map[string]bool{}
	for _, n := range p.Nodes {
		if seen[n.Name] {
			return fmt.Errorf("duplicate node %q", n.Name)
		}
		seen[n.Name] = true
		switch n.Role {
		case RoleWorkstation, RoleControlPlane, RoleWorker:
		default:
			return fmt.Errorf("node %s: unknown role %q", n.Name, n.Role)
		}
		if n.CPUs <= 0 || n.Memory == "" || n.Disk == "" {
			return fmt.Errorf("node %s: cpus, memory and disk are required", n.Name)
		}
	}
	// Both modes install the kubeadm toolchain, so both need a pinned version
	// and a node to be the control plane -- `nodes` because the learner is
	// about to make it one, and graders address it either way.
	if p.Provisioning == ProvisionKubeadm || p.Provisioning == ProvisionNodes {
		if p.Kubernetes.Version == "" {
			return fmt.Errorf("kubernetes.version is required (pin an exact patch, never `latest`)")
		}
		if strings.Contains(p.Kubernetes.Version, "latest") {
			return fmt.Errorf("kubernetes.version must be an exact patch, not %q", p.Kubernetes.Version)
		}
		if p.ControlPlane() == nil {
			return fmt.Errorf("a %s profile needs a control-plane node", p.Provisioning)
		}
	}
	// Only kubeadm init needs the subnets; a `nodes` profile may still declare
	// them so a lab's task text and the profile cannot disagree about the CIDR.
	if p.Provisioning == ProvisionKubeadm {
		if p.Kubernetes.PodSubnet == "" || p.Kubernetes.ServiceSubnet == "" {
			return fmt.Errorf("kubernetes.podSubnet and serviceSubnet are required")
		}
	}
	if p.Kubernetes.PodSubnet != "" && strings.HasPrefix(p.Kubernetes.PodSubnet, trimCIDR(p.Network.Subnet)) {
		return fmt.Errorf("podSubnet %s overlaps the node network %s", p.Kubernetes.PodSubnet, p.Network.Subnet)
	}
	if err := p.validateAPIEndpoint(); err != nil {
		return err
	}
	if p.Network.Lima != "" && p.Network.Subnet == "" {
		return fmt.Errorf("network.subnet is required when network.lima is set")
	}
	return nil
}

// validateIDs is also used at manager boundaries, since callers can construct
// a Profile directly instead of loading and validating content from disk.
func (p *Profile) validateIDs() error {
	if p == nil {
		return fmt.Errorf("environment profile is required")
	}
	if err := config.ValidateID(p.ID); err != nil {
		return fmt.Errorf("profile id: %w", err)
	}
	for _, n := range p.Nodes {
		if err := config.ValidateID(n.Name); err != nil {
			return fmt.Errorf("node name: %w", err)
		}
	}
	return nil
}

// validateAPIEndpoint checks the load-balancer declaration, and insists on one
// as soon as a profile has more than one control plane.
//
// That last rule is the valuable one. Without an endpoint, a three-control-plane
// profile inits with --apiserver-advertise-address only and the second
// `kubeadm join --control-plane` fails twenty minutes into provisioning with an
// error that says nothing about the cause.
func (p *Profile) validateAPIEndpoint() error {
	e := p.APIEndpoint
	declared := e.Node != "" || e.Name != "" || e.Port != 0
	if declared {
		if e.Node == "" || e.Name == "" || e.Port == 0 {
			return fmt.Errorf("apiEndpoint needs node, name and port together")
		}
		n := p.NodeByName(e.Node)
		if n == nil {
			return fmt.Errorf("apiEndpoint.node %q is not a node in this profile", e.Node)
		}
		if n.Role == RoleControlPlane {
			return fmt.Errorf("apiEndpoint.node %q is a control plane; "+
				"put the load balancer on a node that is not one of its own backends", e.Node)
		}
	}
	if len(p.ControlPlanes()) > 1 && !declared {
		return fmt.Errorf("a profile with %d control planes needs an apiEndpoint; "+
			"--control-plane-endpoint is written into the API server certificate at "+
			"`kubeadm init` and cannot be added afterwards", len(p.ControlPlanes()))
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
	if err := config.ValidateID(id); err != nil {
		return nil, fmt.Errorf("environment profile: %w", err)
	}
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
	} else if p.ID != id {
		return nil, fmt.Errorf("%s: profile id %q does not match directory %q", name, p.ID, id)
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
			if err := config.ValidateID(e.Name()); err != nil {
				return nil, fmt.Errorf("environment profile directory: %w", err)
			}
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}
