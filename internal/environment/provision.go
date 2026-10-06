package environment

import (
	"context"
	"errors"
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
	// ControlPlaneEndpoint is "name:port" when the profile puts a load
	// balancer in front of the API server, and "" otherwise. Scripts branch on
	// it being empty, so single-control-plane profiles render unchanged.
	ControlPlaneEndpoint     string
	ControlPlaneEndpointName string
	ControlPlaneEndpointPort int
	// ControlPlaneIPs maps each control-plane node to its address, for the
	// load balancer's backend list.
	ControlPlaneIPs map[string]string
	K8sVersion      string
	K8sMinor        string
	PodSubnet       string
	ServiceSubnet   string
	NodeSubnet      string
	CalicoVersion   string
	MetricsVersion  string
	HelmVersion     string
	// Platform addon versions. See Profile.Addons for why each is installed.
	LocalPathVersion    string
	IngressNginxVersion string
	GatewayAPIVersion   string
	NGFVersion          string
	StudentUser         string
	JoinCommand         string
	Kubeconfig          string
	// ShellDefaults is the .bashrc block terminal.sh installs, read from
	// content so `dojo scrub --defaults` can re-seed the identical text.
	ShellDefaults string
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
	defaults, err := m.ShellDefaults()
	if err != nil {
		return nil, err
	}
	v.ShellDefaults = defaults
	if n := m.Profile.NodeByName(node); n != nil {
		v.Role = n.Role
	}
	if cp != nil {
		v.ControlPlane = cp.Name
		v.ControlPlaneIP = ips[cp.Name]
	}
	if m.Profile.HasAPIEndpoint() {
		v.ControlPlaneEndpoint = m.Profile.APIEndpointAddr()
		v.ControlPlaneEndpointName = m.Profile.APIEndpoint.Name
		v.ControlPlaneEndpointPort = m.Profile.APIEndpoint.Port
	}
	v.ControlPlaneIPs = map[string]string{}
	for _, n := range m.Profile.ControlPlanes() {
		v.ControlPlaneIPs[n.Name] = ips[n.Name]
	}
	return v, nil
}

// ShellDefaultsFile holds the .bashrc block that gives the student `k`, $do
// and $now. It is one file so that provisioning and `dojo scrub --defaults`
// cannot disagree about what the defaults are.
const ShellDefaultsFile = "shell-defaults.bash"

// ShellDefaultsMarker is the line both the provisioning guard and the scrub
// look for to decide whether the block is already installed.
const ShellDefaultsMarker = "# dojo shell defaults"

// ShellDefaults returns the .bashrc block as content ships it.
func (m *Manager) ShellDefaults() (string, error) {
	raw, err := m.readProvisioningFile(ShellDefaultsFile)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(raw), "\n"), nil
}

// readProvisioningFile finds a provisioning file, preferring the profile's own
// copy. Profiles may share the common scripts rather than copying them.
func (m *Manager) readProvisioningFile(name string) ([]byte, error) {
	raw, err := m.Src.Read(path.Join(m.Profile.Dir(), "provisioning", name))
	if err == nil {
		return raw, nil
	}
	raw, err = m.Src.Read(path.Join("environments", "_common", "provisioning", name))
	if err != nil {
		return nil, fmt.Errorf("provisioning file %s not found for profile %s", name, m.Profile.ID)
	}
	return raw, nil
}

// renderScript loads a provisioning template from the content root.
func (m *Manager) renderScript(name string, v *scriptVars) (string, error) {
	raw, err := m.readProvisioningFile(name)
	if err != nil {
		return "", err
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
	if err := m.verifyNodeAddresses(ctx); err != nil {
		return err
	}
	if err := m.baseProvision(ctx, force); err != nil {
		return err
	}
	// A `nodes` profile hands over prepared machines and stops. It still gets
	// verifyNodeAddresses above, because kube-node.sh writes the --node-ip pin
	// that check reads, and a drifted address breaks the cluster the learner
	// builds exactly as it breaks a provisioned one.
	if m.Profile.Provisioning == ProvisionNodes {
		return m.kubeNodeProvision(ctx, force)
	}
	return m.kubeadmProvision(ctx, force)
}

// verifyNodeAddresses refuses to continue when a node's address no longer
// matches the one it was provisioned with.
//
// Everything kubeadm writes embeds the address a node had at `kubeadm init`
// time: the API server certificate's SANs, all four static Pod manifests,
// every kubeconfig in /etc/kubernetes, and each kubelet's `--node-ip`. If an
// address moves afterwards, none of that follows it and the control plane
// cannot come back -- kubelet logs `failed to validate nodeIP: node IP "x" not
// found in the host's network interfaces` and nothing else says why.
//
// Without this check the symptom is a ten-minute wait in WaitReady followed by
// a timeout that names none of the above. Addresses are stable across an
// ordinary stop/start -- Lima derives them from each instance's MAC -- so this
// fires only when something really has changed, and then it says so at once.
func (m *Manager) verifyNodeAddresses(ctx context.Context) error {
	ips, err := m.allIPs(ctx)
	if err != nil {
		return err
	}
	type drift struct{ node, was, now string }
	var moved []drift
	for _, n := range m.Profile.Nodes {
		if n.Role == RoleWorkstation {
			// Not part of the cluster: its kubeconfig and /etc/hosts are
			// rewritten on every Up, so a new address costs it nothing.
			continue
		}
		// /etc/default/kubelet is the address this node was provisioned with,
		// and the one kubelet will insist on. Absent on a first run.
		out, err := m.Run(ctx, n.Name, "cat /etc/default/kubelet 2>/dev/null || true\n")
		if err != nil {
			return err
		}
		was := nodeIPArg(out)
		if was == "" {
			continue
		}
		if now := ips[n.Name]; now != was {
			moved = append(moved, drift{node: n.Name, was: was, now: now})
		}
	}
	if len(moved) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("node addresses have changed since this environment was built:\n")
	for _, d := range moved {
		fmt.Fprintf(&b, "  %s was provisioned as %s and is now %s\n", d.node, d.was, d.now)
	}
	b.WriteString("\nThe cluster's certificates, static Pod manifests and kubeconfigs all\n")
	b.WriteString("embed the old addresses, so it cannot be brought back as it is.\n")
	b.WriteString("Rebuild it with `dojo env reset`.")
	return errors.New(b.String())
}

// nodeIPArg extracts the address from an /etc/default/kubelet body.
//
// The flag is not a separate word: the file holds a single assignment,
// `KUBELET_EXTRA_ARGS=--node-ip=10.0.0.1`, so splitting on whitespace and
// matching a prefix finds nothing. Search for the flag anywhere, then read to
// the next separator so further arguments after it are ignored.
func nodeIPArg(kubeletDefaults string) string {
	_, after, ok := strings.Cut(kubeletDefaults, "--node-ip=")
	if !ok {
		return ""
	}
	after = strings.TrimLeft(after, `"'`)
	if i := strings.IndexAny(after, " \t\r\n\"'"); i >= 0 {
		after = after[:i]
	}
	return after
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
	// The API server endpoint is a name rather than an address precisely so it
	// can be rewritten here. --control-plane-endpoint is baked into the API
	// server certificate's SANs at `kubeadm init` and cannot be changed
	// afterwards; if the load balancer's address ever moves, this line is the
	// whole fix.
	if m.Profile.HasAPIEndpoint() {
		if ip := ips[m.Profile.APIEndpoint.Node]; ip != "" {
			fmt.Fprintf(&block, "%s %s\n", ip, m.Profile.APIEndpoint.Name)
		}
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
// kubeNodeProvision makes every cluster node ready for kubeadm and stops
// there: containerd, the kubeadm toolchain, swap off, the sysctls and the
// --node-ip pin. It is the whole of a `nodes` profile and the first phase of a
// kubeadm one.
func (m *Manager) kubeNodeProvision(ctx context.Context, force bool) error {
	for _, n := range m.Profile.ClusterNodes() {
		if err := m.step(ctx, n.Name, "kube-node.sh", force,
			fmt.Sprintf("installing containerd and Kubernetes %s", m.Profile.Kubernetes.Version)); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) kubeadmProvision(ctx context.Context, force bool) error {
	cp := m.Profile.ControlPlane()
	workers := m.Profile.NodesByRole(RoleWorker)

	if err := m.kubeNodeProvision(ctx, force); err != nil {
		return err
	}

	// The load balancer has to answer before `kubeadm init` runs: the endpoint
	// goes into the API server certificate's SANs and into every kubeconfig at
	// init time. haproxy starts happily with its backends down, which is what
	// resolves the chicken-and-egg.
	if m.Profile.HasAPIEndpoint() {
		if err := m.step(ctx, m.Profile.APIEndpoint.Node, "api-lb.sh", force,
			"installing the API server load balancer"); err != nil {
			return err
		}
	}

	if err := m.step(ctx, cp.Name, "control-plane.sh", force, "running kubeadm init (this takes a few minutes)"); err != nil {
		return err
	}
	if err := m.step(ctx, cp.Name, "cni.sh", force, "installing Calico"); err != nil {
		return err
	}

	// Further control planes join after the CNI, so their Pods can schedule.
	for _, n := range m.Profile.ControlPlanes()[1:] {
		if err := m.joinControlPlane(ctx, n.Name, force); err != nil {
			return err
		}
	}

	// Every control plane carries its own stacked etcd member, so each one
	// needs the tools to inspect it -- and etcd-tools.sh reads the version out
	// of the node's own etcd manifest, so it has to run *after* that node has
	// joined and has one. One iteration on a single-CP profile.
	for _, n := range m.Profile.ControlPlanes() {
		if err := m.step(ctx, n.Name, "etcd-tools.sh", force, "installing etcdctl and etcdutl"); err != nil {
			return err
		}
	}

	for _, w := range workers {
		if err := m.joinWorker(ctx, w.Name, force); err != nil {
			return err
		}
	}

	// Each addon step is skipped when its version is unset, so a profile that
	// exists to exercise the control plane does not spend eight minutes
	// installing an ingress controller no lab on it will use.
	if m.Profile.Addons.MetricsServer != "" {
		if err := m.step(ctx, cp.Name, "addons.sh", force, "installing metrics-server"); err != nil {
			return err
		}
	}
	if m.Profile.Addons.LocalPathProvisioner != "" {
		if err := m.step(ctx, cp.Name, "dynamic-storage.sh", force, "installing the local-path provisioner"); err != nil {
			return err
		}
	}
	if m.Profile.Addons.IngressNginx != "" {
		if err := m.step(ctx, cp.Name, "ingress.sh", force, "installing ingress-nginx and the Gateway API"); err != nil {
			return err
		}
	}
	if err := m.distributeKubeconfig(ctx); err != nil {
		return err
	}
	return m.WaitReady(ctx, 10*time.Minute)
}

// joinControlPlane adds a further control plane to an existing cluster.
//
// Both credentials are minted here rather than read from `kubeadm init`'s
// output: the bootstrap token expires after 24 hours and the certificate key
// after two, and `dojo setup` is resumable across days.
//
// The control-plane-only flags go into the command string rather than into
// join.sh, because that template is shared with joinWorker and
// --apiserver-advertise-address would break a worker join.
func (m *Manager) joinControlPlane(ctx context.Context, node string, force bool) error {
	// A distinct marker from joinWorker's, so a --force rerun cannot confuse
	// the two kinds of join.
	marker := path.Join(markerDir, "join-control-plane")
	joined, err := m.markerExists(ctx, node, marker)
	if err != nil {
		return err
	}
	if joined && !force {
		ui.Detail("%s: already joined as a control plane", node)
		return nil
	}
	ui.Step("%s: joining as a control plane", node)
	first := m.Profile.ControlPlanes()[0]

	join, err := m.Run(ctx, first.Name, "set -euo pipefail\nkubeadm token create --print-join-command\n")
	if err != nil {
		return fmt.Errorf("mint join command on %s: %w", first.Name, err)
	}
	join = strings.TrimSpace(join)
	if !strings.HasPrefix(join, "kubeadm join") {
		return fmt.Errorf("unexpected join command from %s: %q", first.Name, join)
	}

	key, err := m.Run(ctx, first.Name,
		"set -euo pipefail\nkubeadm init phase upload-certs --upload-certs | tail -1\n")
	if err != nil {
		return fmt.Errorf("upload certs on %s: %w", first.Name, err)
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("no certificate key returned by %s", first.Name)
	}

	v, err := m.vars(ctx, node)
	if err != nil {
		return err
	}
	v.JoinCommand = fmt.Sprintf("%s --control-plane --certificate-key %s --apiserver-advertise-address %s",
		join, key, v.NodeIP)
	body, err := m.renderScript("join.sh", v)
	if err != nil {
		return err
	}
	res, err := m.Exec(ctx, node, body, "root")
	if err != nil {
		return fmt.Errorf("kubeadm join --control-plane on %s failed: %w\n%s", node, err, tailOf(res))
	}
	return m.touchMarker(ctx, node, marker)
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
	// Point the workstation at the load balancer when there is one, so a
	// learner's kubectl survives the control plane it happens to be talking to
	// going away -- which is the entire subject of an HA lab.
	server := ""
	if m.Profile.HasAPIEndpoint() {
		server = "https://" + m.Profile.APIEndpointAddr()
	} else {
		cpIP, err := m.nodeIP(ctx, cp.Name)
		if err != nil {
			return err
		}
		// kubeadm writes the advertise address already, but rewriting it keeps
		// this correct if a profile ever advertises a different endpoint.
		server = "https://" + cpIP + ":6443"
	}
	kubeconfig := rewriteServer(string(raw), server)
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
	// Every control plane, not just the first: a four-node HA profile that
	// counted 1 + workers would call the cluster ready at two.
	want := len(m.Profile.ControlPlanes()) + len(m.Profile.NodesByRole(RoleWorker))
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
