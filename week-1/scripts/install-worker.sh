#!/bin/bash
set -euxo pipefail

K8S_VERSION="${1:-1.33}"

source /tmp/common.sh "$K8S_VERSION"

if [ -f /etc/kubernetes/kubelet.conf ]; then
  echo "This node has already joined a cluster — skipping."
  exit 0
fi

sudo bash /tmp/join-command.sh
