variable "node_count" {
  description = "Number of VMs to provision (1 for now, 2 later for the k3s cluster)"
  type        = number
  default     = 1
}

variable "image_name" {
  description = "Glance image to boot from"
  type        = string
  default     = "ubuntu-cloud"
}

variable "flavor_name" {
  description = "OpenStack flavor (CPU/RAM/disk size)"
  type        = string
  default     = "m1.medium"
}

variable "external_network_name" {
  description = "The provider/external network for floating IPs"
  type        = string
  default     = "public"
}

variable "key_pair_public_key_path" {
  description = "Path to the local SSH public key to import into OpenStack"
  type        = string
  default     = "~/.ssh/tf_key.pub"
}

variable "k8s_minor_version" {
  description = "Kubernetes minor version (matches pkgs.k8s.io repo path)"
  type        = string
  default     = "1.33"
}


variable "key_pair_private_key_path" {
  description = "Local path to the private key matching key_pair_public_key_path"
  type        = string
  default     = "~/.ssh/tf_key"
}
