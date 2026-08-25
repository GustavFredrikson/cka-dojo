#!/usr/bin/env bash
# containerd + kubelet/kubeadm/kubectl on a machine that will be part of the
# cluster. We install this ourselves rather than using a prebuilt Kubernetes
# image, because labs need to break the installation itself.
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
APT="apt-get -y -o DPkg::Lock::Timeout=900 -o Acquire::Retries=3"

swapoff -a
sed -i '/\sswap\s/ s/^\(.*\)$/#\1/g' /etc/fstab

cat > /etc/modules-load.d/k8s.conf <<'MODULES'
overlay
br_netfilter
MODULES
modprobe overlay
modprobe br_netfilter

cat > /etc/sysctl.d/99-kubernetes.conf <<'SYSCTL'
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
SYSCTL
sysctl --system >/dev/null

install -m 0755 -d /etc/apt/keyrings

# containerd from Docker's repository: current releases, and the same source
# the upstream Kubernetes install docs point at.
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  > /etc/apt/sources.list.d/docker.list

curl -fsSL "https://pkgs.k8s.io/core:/stable:/v{{.K8sMinor}}/deb/Release.key" \
  | gpg --dearmor --yes -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
chmod 0644 /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v{{.K8sMinor}}/deb/ /" \
  > /etc/apt/sources.list.d/kubernetes.list

$APT update -qq
$APT install -qq containerd.io

# Exact patch pin. Drifting silently would make a "check the version" lab lie.
PKG="$(apt-cache madison kubeadm | awk '{print $3}' | grep -m1 '^{{.K8sVersion}}-' || true)"
if [ -z "$PKG" ]; then
  echo "Kubernetes {{.K8sVersion}} is not in the v{{.K8sMinor}} apt repo. Available:" >&2
  apt-cache madison kubeadm | awk '{print "  " $3}' | head -20 >&2
  echo "Update kubernetes.version in the environment profile." >&2
  exit 1
fi
$APT install -qq kubelet="$PKG" kubeadm="$PKG" kubectl="$PKG"
apt-mark hold kubelet kubeadm kubectl containerd.io >/dev/null

# containerd must use the systemd cgroup driver, like kubelet does.
mkdir -p /etc/containerd
containerd config default > /etc/containerd/config.toml
sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml

# Keep the sandbox image in step with what kubeadm expects. The key name for
# this differs between containerd 1.x and 2.x, so match on the value instead.
PAUSE="$(kubeadm config images list --kubernetes-version 'v{{.K8sVersion}}' 2>/dev/null | grep -m1 '/pause:' || true)"
if [ -n "$PAUSE" ]; then
  sed -i -E "s#[A-Za-z0-9./_-]+/pause:[0-9][0-9.]*#${PAUSE}#g" /etc/containerd/config.toml
fi

systemctl daemon-reload
systemctl enable --now containerd
systemctl restart containerd

cat > /etc/crictl.yaml <<'CRICTL'
runtime-endpoint: unix:///run/containerd/containerd.sock
image-endpoint: unix:///run/containerd/containerd.sock
timeout: 10
CRICTL

# Every Lima VM has the same address (192.168.5.15) on its default-route
# interface. Without an explicit node IP, every node would register with an
# identical address and the cluster would be nonsense.
cat > /etc/default/kubelet <<'KUBELET'
KUBELET_EXTRA_ARGS=--node-ip={{.NodeIP}}
KUBELET
sed -i 's|^KUBELET_EXTRA_ARGS=.*|KUBELET_EXTRA_ARGS=--node-ip={{.NodeIP}}|' /etc/default/kubelet

systemctl enable kubelet
echo "{{.Node}} ready for kubeadm: $(kubeadm version -o short), containerd $(containerd --version | awk '{print $3}')"
