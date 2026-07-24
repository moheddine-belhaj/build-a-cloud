#!/bin/bash
set -euxo pipefail

K8S_VERSION="${1:-1.33}"

# Disable swap (kubelet refuses to start with swap enabled)
sudo swapoff -a
sudo sed -i '/ swap / s/^/#/' /etc/fstab

# Kernel modules + sysctl networking required by the CNI plugin
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF
sudo modprobe overlay
sudo modprobe br_netfilter

cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sudo sysctl --system

# Container runtime
sudo apt-get update
sudo apt-get install -y containerd
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml
sudo systemctl restart containerd
sudo systemctl enable containerd

# kubeadm, kubelet, kubectl from the official repo
sudo apt-get install -y apt-transport-https ca-certificates curl gpg
sudo mkdir -p /etc/apt/keyrings
curl -fsSL "https://pkgs.k8s.io/core:/stable:/v${K8S_VERSION}/deb/Release.key" | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v${K8S_VERSION}/deb/ /" | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update
sudo apt-get install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl

# Bootstrap the control plane
for i in 1 2 3; do
  sudo kubeadm init --pod-network-cidr=10.244.0.0/16 && break
  echo "kubeadm init attempt $i failed, resetting and retrying..."
  sudo kubeadm reset -f
  sleep 15
done

# kubeconfig for the ubuntu user
mkdir -p /home/ubuntu/.kube
sudo cp -i /etc/kubernetes/admin.conf /home/ubuntu/.kube/config
sudo chown ubuntu:ubuntu /home/ubuntu/.kube/config

# CNI plugin (Flannel — matches the pod-network-cidr above)
export KUBECONFIG=/home/ubuntu/.kube/config
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

# Single-node only: allow pods to schedule on the control-plane node
kubectl taint nodes --all node-role.kubernetes.io/control-plane- || true

# Save the join command for the two-node task later
sudo kubeadm token create --print-join-command > /home/ubuntu/join-command.sh
sudo chmod +x /home/ubuntu/join-command.sh
sudo chown ubuntu:ubuntu /home/ubuntu/join-command.sh
