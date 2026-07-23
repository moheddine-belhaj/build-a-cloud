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
  default     = "m1.small"
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
