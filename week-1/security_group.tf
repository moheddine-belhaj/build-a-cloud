resource "openstack_networking_secgroup_v2" "tf_sg" {
  name        = "tf-sg"
  description = "Security group for Terraform-managed instances"
}

resource "openstack_networking_secgroup_rule_v2" "tf_sg_ssh" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 22
  port_range_max    = 22
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = openstack_networking_secgroup_v2.tf_sg.id
}

resource "openstack_networking_secgroup_rule_v2" "tf_sg_k8s_api" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 6443
  port_range_max    = 6443
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = openstack_networking_secgroup_v2.tf_sg.id
}

resource "openstack_networking_secgroup_rule_v2" "tf_sg_internal" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 1
  port_range_max    = 65535
  remote_group_id   = openstack_networking_secgroup_v2.tf_sg.id
  security_group_id = openstack_networking_secgroup_v2.tf_sg.id
}

resource "openstack_networking_secgroup_rule_v2" "tf_sg_icmp" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "icmp"
  security_group_id = openstack_networking_secgroup_v2.tf_sg.id
}
