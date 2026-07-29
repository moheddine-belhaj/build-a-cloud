output "kubeconfig" {
  value     = stackit_ske_kubeconfig.kubeconfig.kube_config
  sensitive = true
}