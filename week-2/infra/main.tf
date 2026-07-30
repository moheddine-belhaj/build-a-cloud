# Resource: StackIT SKE cluster
# Creates a Kubernetes cluster named "paas-ske" with a single node pool.
# - `kubernetes_version_min`: Minimum K8s version
# - `node_pools`: Defines machine type, scaling and storage
# retrigger deploy 3

resource "stackit_ske_cluster" "ske_cluster" {
  project_id             = var.project_id
  name                   = "paas-ske"
  kubernetes_version_min = "1.34.9"

  node_pools = [{
    name               = "default"
    machine_type       = "g1a.2d"
    minimum            = 2
    maximum            = 3
    availability_zones = ["eu01-1", "eu01-2", "eu01-3"]
    os_name            = "flatcar"
    os_version_min     = "4593.2.2"
    volume_size        = 32
    volume_type        = "storage_premium_perf6"
  }]

  maintenance = {
    enable_kubernetes_version_updates    = true
    enable_machine_image_version_updates = true
    start = "01:00:00Z"
    end   = "02:00:00Z"
  }
}

# Resource: SKE kubeconfig
# Generates and refreshes the kubeconfig for the created cluster so `kubectl` can connect.
# Kubeconfig expiration → 8 hours, in seconds (8 * 3600) = 28800
resource "stackit_ske_kubeconfig" "kubeconfig" {
  project_id   = var.project_id
  cluster_name = stackit_ske_cluster.ske_cluster.name
  refresh      = true
  expiration   = 28800
}