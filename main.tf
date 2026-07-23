resource "openstack_compute_keypair_v2" "tf_key" {
  name       = "tf-key"
  public_key = file(var.key_pair_public_key_path)
}

resource "openstack_compute_instance_v2" "tf_vm" {
  count       = var.node_count
  name        = "tf-vm-${count.index + 1}"
  image_name  = var.image_name
  flavor_name = var.flavor_name
  key_pair    = openstack_compute_keypair_v2.tf_key.name

  security_groups = [openstack_networking_secgroup_v2.tf_sg.name]

  network {
    uuid = openstack_networking_network_v2.tf_net.id
  }

  depends_on = [openstack_networking_router_interface_v2.tf_router_interface]
}

resource "openstack_networking_floatingip_v2" "tf_fip" {
  count   = var.node_count
  pool    = var.external_network_name
}

data "openstack_networking_port_v2" "tf_vm_port" {
  count      = var.node_count
  device_id  = openstack_compute_instance_v2.tf_vm[count.index].id
  network_id = openstack_networking_network_v2.tf_net.id
}

resource "openstack_networking_floatingip_associate_v2" "tf_fip_assoc" {
  count       = var.node_count
  floating_ip = openstack_networking_floatingip_v2.tf_fip[count.index].address
  port_id     = data.openstack_networking_port_v2.tf_vm_port[count.index].id
}
