#!/usr/bin/env bash
# Base packages and the student account. Runs on every node of every profile.
#
# Rendered as a Go template on the host, so template values are already
# substituted by the time bash sees this file.
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive

# Cloud images run unattended-upgrades on first boot and hold the dpkg lock;
# waiting is cheaper than racing it.
APT="apt-get -y -o DPkg::Lock::Timeout=900 -o Acquire::Retries=3"

$APT update -qq
$APT install -qq --no-install-recommends \
  ca-certificates curl wget gnupg apt-transport-https \
  vim nano less jq tree bash-completion \
  openssh-client iproute2 iputils-ping bind9-dnsutils netcat-openbsd \
  lsof psmisc tcpdump

# The learner account. Everything a lab asks for is done as this user, so that
# SSH between nodes and the shell prompt match a real exam workstation.
id {{.StudentUser}} >/dev/null 2>&1 || useradd -m -s /bin/bash {{.StudentUser}}
printf '%s ALL=(ALL) NOPASSWD:ALL\n' '{{.StudentUser}}' > /etc/sudoers.d/90-dojo-student
chmod 0440 /etc/sudoers.d/90-dojo-student

cat > /etc/profile.d/dojo.sh <<'PROFILE'
export EDITOR=vim
export KUBE_EDITOR=vim
PROFILE

install -d -o {{.StudentUser}} -g {{.StudentUser}} -m 0700 /home/{{.StudentUser}}/.ssh
echo "base packages ready on {{.Node}}"
