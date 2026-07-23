resource "openstack_networking_network_v2" "tf_net" {
  name           = "tf-net"
  admin_state_up = true
}

resource "openstack_networking_subnet_v2" "tf_subnet" {
  name            = "tf-subnet"
  network_id      = openstack_networking_network_v2.tf_net.id
  cidr            = "192.168.60.0/24"
  ip_version      = 4
  dns_nameservers = ["8.8.8.8"]
}

data "openstack_networking_network_v2" "external" {
  name = var.external_network_name
}

resource "openstack_networking_router_v2" "tf_router" {
  name                = "tf-router"
  admin_state_up      = true
  external_network_id = data.openstack_networking_network_v2.external.id
}

resource "openstack_networking_router_interface_v2" "tf_router_interface" {
  router_id = openstack_networking_router_v2.tf_router.id
  subnet_id = openstack_networking_subnet_v2.tf_subnet.id
}
