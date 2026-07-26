#!/bin/bash
set -euxo pipefail

K8S_VERSION="${1:-1.33}"

source /tmp/common.sh "$K8S_VERSION"

if [ -f /etc/kubernetes/admin.conf ]; then
  echo "Control plane already initialized — skipping."
  exit 0
fi

for i in 1 2 3; do
  sudo kubeadm init --pod-network-cidr=10.244.0.0/16 && break
  echo "kubeadm init attempt $i failed, resetting and retrying..."
  sudo kubeadm reset -f
  sleep 15
done

mkdir -p /home/ubuntu/.kube
sudo cp -i /etc/kubernetes/admin.conf /home/ubuntu/.kube/config
sudo chown ubuntu:ubuntu /home/ubuntu/.kube/config

export KUBECONFIG=/home/ubuntu/.kube/config
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

sudo kubeadm token create --print-join-command > /home/ubuntu/join-command.sh
sudo chmod +x /home/ubuntu/join-command.sh
sudo chown ubuntu:ubuntu /home/ubuntu/join-command.sh
