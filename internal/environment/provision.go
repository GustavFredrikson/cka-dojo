package environment

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/gustavfredrikson/cka-dojo/internal/config"
	"github.com/gustavfredrikson/cka-dojo/internal/provider"
	"github.com/gustavfredrikson/cka-dojo/internal/ui"
)

// markerDir holds one file per completed provisioning step, so `dojo setup`
// is resumable: a failure twenty minutes in does not redo the first twenty.
const markerDir = "/var/lib/dojo/steps"

// scriptVars is the data every provisioning template is rendered against.
type scriptVars struct {
	Node           string
	Role           string
	NodeIP         string
	NodeIPs        map[string]string
	ControlPlane   string
	ControlPlaneIP string
	K8sVersion     string
	K8sMinor       string
	PodSubnet      string
	ServiceSubnet  string
	NodeSubnet     string
	CalicoVersion  string
	MetricsVersion string
	HelmVersion    string
	// Platform addon versions. See Profile.Addons for why each is installed.
	LocalPathVersion    string
	IngressNginxVersion string
	GatewayAPIVersion   string
	NGFVersion          string
	StudentUser         string
	JoinCommand         string
	Kubeconfig          string
}

func (m *Manager) vars(ctx context.Context, node string) (*scriptVars, error) {
	ips, err := m.allIPs(ctx)
	if err != nil {
		return nil, err
	}
	cp := m.Profile.ControlPlane()
	v := &scriptVars{
		Node:           node,
		NodeIP:         ips[node],
		NodeIPs:        ips,
		K8sVersion:     m.Profile.Kubernetes.Version,
		K8sMinor:       m.Profile.Minor(),
		PodSubnet:      m.Profile.Kubernetes.PodSubnet,
		ServiceSubnet:  m.Profile.Kubernetes.ServiceSubnet,
		NodeSubnet:     m.Profile.Network.Subnet,
		CalicoVersion:  m.Profile.CNI.Version,
		MetricsVersion: m.Profile.Addons.MetricsServer,
		HelmVersion:    m.Profile.Addons.Helm,

		LocalPathVersion:    m.Profile.Addons.LocalPathProvisioner,
		IngressNginxVersion: m.Profile.Addons.IngressNginx,
		GatewayAPIVersion:   m.Profile.Addons.GatewayAPI,
		NGFVersion:          m.Profile.Addons.NginxGatewayFabric,
		StudentUser:         StudentUser,
		Kubeconfig:          AdminKubeconfig,
	}
	if n := m.Profile.NodeByName(node); n != nil {
		v.Role = n.Role
	}
	if cp != nil {
		v.ControlPlane = cp.Name
		v.ControlPlaneIP = ips[cp.Name]
	}
	return v, nil
}

// renderScript loads a provisioning template from the content root.
func (m *Manager) renderScript(name string, v *scriptVars) (string, error) {
	raw, err := m.Src.Read(path.Join(m.Profile.Dir(), "provisioning", name))
	if err != nil {
		// Profiles may share the common scripts rather than copying them.
		raw, err = m.Src.Read(path.Join("environments", "_common", "provisioning", name))
		if err != nil {
			return "", fmt.Errorf("provisioning script %s not found for profile %s", name, m.Profile.ID)
		}
	}
	tpl, err := template.New(name).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", name, err)
	}
	var sb strings.Builder
	if err := tpl.Execute(&sb, v); err != nil {
		return "", fmt.Errorf("render %s: %w", name, err)
	}
	return sb.String(), nil
}

// step runs a provisioning script once per node, guarded by a marker file.
func (m *Manager) step(ctx context.Context, node, script string, force bool, label string) error {
	marker := path.Join(markerDir, strings.TrimSuffix(script, ".sh"))
	if !force {
		done, err := m.markerExists(ctx, node, marker)
		if err != nil {
			return err
		}
		if done {
			ui.Detail("%s: %s already done", node, script)
			return nil
		}
	}
	v, err := m.vars(ctx, node)
	if err != nil {
		return err
	}
	body, err := m.renderScript(script, v)
	if err != nil {
		return err
	}
	if label != "" {
		ui.Step("%s: %s", node, label)
	}
	start := time.Now()
	res, err := m.Exec(ctx, node, body, "root")
	if err != nil {
		return fmt.Errorf("%s on %s failed: %w\n%s", script, node, err, tailOf(res))
	}
	ui.Detail("%s: %s finished in %s", node, script, time.Since(start).Round(time.Second))
	return m.touchMarker(ctx, node, marker)
}

func tailOf(res provider.ExecResult) string {
	out := res.Stderr
	if strings.TrimSpace(out) == "" {
		out = res.Stdout
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	return strings.Join(lines, "\n")
}

func (m *Manager) markerExists(ctx context.Context, node, marker string) (bool, error) {
	script := fmt.Sprintf("test -f %q && echo yes || echo no\n", marker)
	out, err := m.Run(ctx, node, script)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "yes", nil
}

func (m *Manager) touchMarker(ctx context.Context, node, marker string) error {
	_, err := m.Run(ctx, node, fmt.Sprintf("mkdir -p %q && touch %q\n", markerDir, marker))
	return err
}

// Up brings the whole environment to its ready state. It is idempotent and
// resumable, and safe to call before every lab.
func (m *Manager) Up(ctx context.Context, force bool) error {
	if err := m.EnsureNodes(ctx); err != nil {
		return err
	}
	if m.Profile.Provisioning == ProvisionNone {
		return m.baseProvision(ctx, force)
	}
	if err := m.baseProvision(ctx, force); err != nil {
		return err
	}
	return m.kubeadmProvision(ctx, force)
}

// baseProvision does the work every profile needs: name resolution between
// nodes, the student account, SSH between machines, and base packages.
func (m *Manager) baseProvision(ctx context.Context, force bool) error {
	ui.Step("discovering node addresses on %s", m.Profile.Network.Subnet)
	ips, err := m.allIPs(ctx)
	if err != nil {
		return err
	}
	for _, n := range sortedKeys(ips) {
		ui.Detail("%s = %s", n, ips[n])
	}

	for _, n := range m.Profile.Nodes {
		if err := m.step(ctx, n.Name, "common.sh", force, "installing base packages and the student account"); err != nil {
			return err
		}
	}
	if err := m.writeHosts(ctx, ips); err != nil {
		return err
	}
	if err := m.setupSSH(ctx); err != nil {
		return err
	}
	for _, n := range m.Profile.Nodes {
		if n.Role != RoleWorkstation {
			continue
		}
		if err := m.step(ctx, n.Name, "terminal.sh", force, "installing kubectl and helm"); err != nil {
			return err
		}
	}
	return nil
}

// writeHosts gives every node short-name resolution for every other node. We
// own /etc/hosts rather than relying on provider DNS so a lab can rely on
// `ssh worker1` and on node names matching hostnames.
func (m *Manager) writeHosts(ctx context.Context, ips map[string]string) error {
	var block strings.Builder
	block.WriteString("# BEGIN dojo\n")
	for _, name := range sortedKeys(ips) {
		fmt.Fprintf(&block, "%s %s\n", ips[name], name)
	}
	block.WriteString("# END dojo\n")

	script := fmt.Sprintf(`set -euo pipefail
sed -i '/# BEGIN dojo/,/# END dojo/d' /etc/hosts
cat >> /etc/hosts <<'DOJO_HOSTS_EOF'
%sDOJO_HOSTS_EOF
`, block.String())

	ui.Step("writing /etc/hosts on all nodes")
	for _, n := range m.Profile.Nodes {
		if _, err := m.Run(ctx, n.Name, script); err != nil {
			return fmt.Errorf("write /etc/hosts on %s: %w", n.Name, err)
		}
	}
	return nil
}

// setupSSH puts a dedicated keypair on the workstation and authorises it on
// every node, so `ssh cp1` works the way it does on the real exam desktop.
func (m *Manager) setupSSH(ctx context.Context) error {
	ws := m.Profile.Workstation()
	if ws == nil {
		return nil
	}
	dir, err := config.EnvDir(m.Profile.ID)
	if err != nil {
		return err
	}
	keyPath := filepath.Join(dir, "id_ed25519")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		ui.Step("generating an SSH key for the workstation")
		cmd := exec.CommandContext(ctx, "ssh-keygen", "-t", "ed25519", "-N", "", "-C", "dojo-"+m.Profile.ID, "-f", keyPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ssh-keygen: %w: %s", err, strings.TrimSpace(string(out)))
		}
	}
	priv, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	pub, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return err
	}

	authScript := fmt.Sprintf(`set -euo pipefail
install -d -o %[1]s -g %[1]s -m 700 /home/%[1]s/.ssh
grep -qxF %[2]q /home/%[1]s/.ssh/authorized_keys 2>/dev/null || echo %[2]q >> /home/%[1]s/.ssh/authorized_keys
chown %[1]s:%[1]s /home/%[1]s/.ssh/authorized_keys
chmod 600 /home/%[1]s/.ssh/authorized_keys
`, StudentUser, strings.TrimSpace(string(pub)))

	ui.Step("authorising the workstation key on every node")
	for _, n := range m.Profile.Nodes {
		if _, err := m.Run(ctx, n.Name, authScript); err != nil {
			return fmt.Errorf("authorize key on %s: %w", n.Name, err)
		}
	}

	var hosts []string
	for _, n := range m.Profile.Nodes {
		if n.Name != ws.Name {
			hosts = append(hosts, n.Name)
		}
	}
	// Host keys change every time an environment is rebuilt, so pinning them
	// would only produce scary warnings during normal use.
	sshConfig := fmt.Sprintf(`Host %s
  User %s
  IdentityFile ~/.ssh/id_ed25519
  StrictHostKeyChecking no
  UserKnownHostsFile /dev/null
  LogLevel ERROR
`, strings.Join(hosts, " "), StudentUser)

	vm := m.VMName(ws.Name)
	home := "/home/" + StudentUser
	if err := m.Prov.WriteFile(ctx, vm, home+"/.ssh/id_ed25519", priv, 0o600); err != nil {
		return err
	}
	if err := m.Prov.WriteFile(ctx, vm, home+"/.ssh/config", []byte(sshConfig), 0o600); err != nil {
		return err
	}
	_, err = m.Run(ctx, ws.Name, fmt.Sprintf("chown -R %[1]s:%[1]s /home/%[1]s/.ssh\n", StudentUser))
	return err
}

// kubeadmProvision installs Kubernetes: nodes, control plane, CNI, workers.
func (m *Manager) kubeadmProvision(ctx context.Context, force bool) error {
	cp := m.Profile.ControlPlane()
	workers := m.Profile.NodesByRole(RoleWorker)

	kubeNodes := append([]*NodeConfig{cp}, workers...)
	for _, n := range kubeNodes {
		if err := m.step(ctx, n.Name, "kube-node.sh", force,
			fmt.Sprintf("installing containerd and Kubernetes %s", m.Profile.Kubernetes.Version)); err != nil {
			return err
		}
	}

	if err := m.step(ctx, cp.Name, "control-plane.sh", force, "running kubeadm init (this takes a few minutes)"); err != nil {
		return err
	}
	if err := m.step(ctx, cp.Name, "cni.sh", force, "installing Calico"); err != nil {
		return err
	}

	for _, w := range workers {
		if err := m.joinWorker(ctx, w.Name, force); err != nil {
			return err
		}
	}

	if err := m.step(ctx, cp.Name, "addons.sh", force, "installing metrics-server"); err != nil {
		return err
	}
	if err := m.step(ctx, cp.Name, "dynamic-storage.sh", force, "installing the local-path provisioner"); err != nil {
		return err
	}
	if err := m.step(ctx, cp.Name, "ingress.sh", force, "installing ingress-nginx and the Gateway API"); err != nil {
		return err
	}
	if err := m.distributeKubeconfig(ctx); err != nil {
		return err
	}
	return m.WaitReady(ctx, 10*time.Minute)
}

// joinWorker fetches a fresh join command from the control plane. Tokens
// expire, so it is always minted at join time rather than cached.
func (m *Manager) joinWorker(ctx context.Context, node string, force bool) error {
	joined, err := m.markerExists(ctx, node, path.Join(markerDir, "join"))
	if err != nil {
		return err
	}
	if joined && !force {
		ui.Detail("%s: already joined", node)
		return nil
	}
	ui.Step("%s: joining the cluster", node)
	cp := m.Profile.ControlPlane()
	join, err := m.Run(ctx, cp.Name, "set -euo pipefail\nkubeadm token create --print-join-command\n")
	if err != nil {
		return fmt.Errorf("mint join command on %s: %w", cp.Name, err)
	}
	join = strings.TrimSpace(join)
	if !strings.HasPrefix(join, "kubeadm join") {
		return fmt.Errorf("unexpected join command from %s: %q", cp.Name, join)
	}
	v, err := m.vars(ctx, node)
	if err != nil {
		return err
	}
	v.JoinCommand = join
	body, err := m.renderScript("join.sh", v)
	if err != nil {
		return err
	}
	res, err := m.Exec(ctx, node, body, "root")
	if err != nil {
		return fmt.Errorf("kubeadm join on %s failed: %w\n%s", node, err, tailOf(res))
	}
	return m.touchMarker(ctx, node, path.Join(markerDir, "join"))
}

// distributeKubeconfig gives the student a cluster-admin kubeconfig on the
// workstation, pointed at the control plane's address on the node network.
func (m *Manager) distributeKubeconfig(ctx context.Context) error {
	ws := m.Profile.Workstation()
	cp := m.Profile.ControlPlane()
	if ws == nil || cp == nil {
		return nil
	}
	ui.Step("installing the student kubeconfig on %s", ws.Name)
	raw, err := m.Prov.ReadFile(ctx, m.VMName(cp.Name), AdminKubeconfig)
	if err != nil {
		return fmt.Errorf("read admin.conf from %s: %w", cp.Name, err)
	}
	cpIP, err := m.nodeIP(ctx, cp.Name)
	if err != nil {
		return err
	}
	// kubeadm writes the advertise address already, but rewriting it keeps
	// this correct if a profile ever advertises a different endpoint.
	kubeconfig := rewriteServer(string(raw), "https://"+cpIP+":6443")
	home := "/home/" + StudentUser
	vm := m.VMName(ws.Name)
	if err := m.Prov.WriteFile(ctx, vm, home+"/.kube/config", []byte(kubeconfig), 0o600); err != nil {
		return err
	}
	_, err = m.Run(ctx, ws.Name, fmt.Sprintf("chown -R %[1]s:%[1]s /home/%[1]s/.kube\n", StudentUser))
	return err
}

func rewriteServer(kubeconfig, server string) string {
	out := make([]string, 0, 32)
	for _, line := range strings.Split(kubeconfig, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "server:") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			out = append(out, indent+"server: "+server)
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// WaitReady blocks until every node registers Ready.
func (m *Manager) WaitReady(ctx context.Context, timeout time.Duration) error {
	cp := m.Profile.ControlPlane()
	if cp == nil {
		return nil
	}
	want := 1 + len(m.Profile.NodesByRole(RoleWorker))
	ui.Step("waiting for %d nodes to become Ready", want)
	deadline := time.Now().Add(timeout)
	for {
		out, err := m.Kubectl(ctx, "get", "nodes", "--no-headers")
		if err == nil {
			ready := 0
			for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
				f := strings.Fields(line)
				if len(f) >= 2 && f[1] == "Ready" {
					ready++
				}
			}
			ui.Detail("%d/%d nodes Ready", ready, want)
			if ready >= want {
				ui.OK("cluster is ready (%d nodes)", ready)
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %d Ready nodes; try `dojo env status` or `dojo setup --verbose`", want)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
