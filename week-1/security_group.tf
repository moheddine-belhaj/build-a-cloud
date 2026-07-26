# Create the security group for the cluster nodes.
resource "openstack_networking_secgroup_v2" "tf_sg" {
  name        = "tf-sg"
  description = "Security group for Terraform-managed instances"
}

# Allow SSH access to the instances from the internet.
resource "openstack_networking_secgroup_rule_v2" "tf_sg_ssh" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 22
  port_range_max    = 22
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = openstack_networking_secgroup_v2.tf_sg.id
}

# Allow Kubernetes API access from the internet.
resource "openstack_networking_secgroup_rule_v2" "tf_sg_k8s_api" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 6443
  port_range_max    = 6443
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = openstack_networking_secgroup_v2.tf_sg.id
}

# Allow full internal TCP traffic between instances in the same security group.
resource "openstack_networking_secgroup_rule_v2" "tf_sg_internal" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 1
  port_range_max    = 65535
  remote_group_id   = openstack_networking_secgroup_v2.tf_sg.id
  security_group_id = openstack_networking_secgroup_v2.tf_sg.id
}

# Allow ICMP traffic for connectivity testing.
resource "openstack_networking_secgroup_rule_v2" "tf_sg_icmp" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "icmp"
  security_group_id = openstack_networking_secgroup_v2.tf_sg.id
}
