output "instance_names" {
  description = "Names of the provisioned instances"
  value       = openstack_compute_instance_v2.tf_vm[*].name
}

output "floating_ips" {
  description = "Public floating IPs assigned to each instance"
  value       = openstack_networking_floatingip_v2.tf_fip[*].address
}

output "private_ips" {
  description = "Private IPs on tf-net for each instance"
  value       = openstack_compute_instance_v2.tf_vm[*].access_ip_v4
}
