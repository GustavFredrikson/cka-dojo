#!/usr/bin/env bash
# The workstation: kubectl, helm, completions. No kubelet, no container
# runtime -- this machine is never part of the cluster, which is what makes
# "the control plane is broken" labs realistic.
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
APT="apt-get -y -o DPkg::Lock::Timeout=900 -o Acquire::Retries=3"

install -m 0755 -d /etc/apt/keyrings
curl -fsSL "https://pkgs.k8s.io/core:/stable:/v{{.K8sMinor}}/deb/Release.key" \
  | gpg --dearmor --yes -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
chmod 0644 /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v{{.K8sMinor}}/deb/ /" \
  > /etc/apt/sources.list.d/kubernetes.list
# Redirection inherits the provisioning umask (077), and an apt list the
# student cannot read makes Ubuntu's command-not-found handler print a
# permission warning over every mistyped command.
chmod 0644 /etc/apt/sources.list.d/kubernetes.list

$APT update -qq

# Pin kubectl to the cluster's exact patch. If the pin is gone from the repo,
# say so loudly rather than silently drifting to another version.
PKG="$(apt-cache madison kubectl | awk '{print $3}' | grep -m1 '^{{.K8sVersion}}-' || true)"
if [ -z "$PKG" ]; then
  echo "kubectl {{.K8sVersion}} is not in the v{{.K8sMinor}} apt repo. Available:" >&2
  apt-cache madison kubectl | awk '{print "  " $3}' | head -20 >&2
  echo "Update kubernetes.version in the environment profile." >&2
  exit 1
fi
$APT install -qq kubectl="$PKG"
apt-mark hold kubectl >/dev/null

ARCH="$(dpkg --print-architecture)"
if ! command -v helm >/dev/null 2>&1; then
  curl -fsSL -o /tmp/helm.tgz "https://get.helm.sh/helm-{{.HelmVersion}}-linux-${ARCH}.tar.gz"
  tar -xzf /tmp/helm.tgz -C /tmp
  install -m 0755 "/tmp/linux-${ARCH}/helm" /usr/local/bin/helm
  rm -rf /tmp/helm.tgz "/tmp/linux-${ARCH}"
fi

# Redirection here inherits the provisioning shell's umask (077), which would
# leave these unreadable by the student -- bash-completion silently skips a
# file it cannot read, and the alias completion below then points at a
# function that was never defined.
kubectl completion bash > /etc/bash_completion.d/kubectl
helm completion bash > /etc/bash_completion.d/helm
chmod 0644 /etc/bash_completion.d/kubectl /etc/bash_completion.d/helm

# The aliases every CKA guide tells you to set up on minute one. The block
# lives in shell-defaults.bash so that `dojo scrub --defaults` re-seeds the
# same text this script installs, rather than a copy that drifts from it.
BASHRC=/home/{{.StudentUser}}/.bashrc
if ! grep -q 'dojo shell defaults' "$BASHRC" 2>/dev/null; then
  cat >> "$BASHRC" <<'BASHRC_EOF'

{{.ShellDefaults}}
BASHRC_EOF
fi
chown {{.StudentUser}}:{{.StudentUser}} "$BASHRC"

echo "workstation ready: $(kubectl version --client=true -o json | jq -r .clientVersion.gitVersion), $(helm version --short)"
