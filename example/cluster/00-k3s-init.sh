#!/bin/bash

set -eo pipefail
OS=$(uname -s | tr A-Z a-z)
ARCH=$(uname -m | sed 's/aarch64/arm64/' | sed 's/x86_64/amd64/')

sudo apt install -y curl open-iscsi nfs-common

# If cluster API is accessed via public IP eg when running on a cloud VM
# Or set it to local IP
PUBLIC_IP=$(curl -s https://api.ipify.org)

install_k3s() {
  if [ ! -f "$KUBECONFIG" ]; then
    curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC=" \
      server \
      --disable=traefik \
      --node-external-ip $PUBLIC_IP \
      --kubelet-arg=allowed-unsafe-sysctls=net.ipv4.ip_forward,net.ipv4.conf.all.src_valid_mark,net.ipv6.conf.all.forwarding \
    " sh -
  fi
}

install_k3s