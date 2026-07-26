resource "openstack_compute_keypair_v2" "tf_key" {
  name       = "tf-key"
  public_key = file(var.key_pair_public_key_path)
}

resource "openstack_compute_instance_v2" "tf_vm" {
  count       = var.node_count
  name        = "tf-vm-${count.index + 1}"
  image_name  = var.image_name
  flavor_name = count.index == 0 ? var.control_plane_flavor_name : var.worker_flavor_name
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

resource "null_resource" "k8s_control_plane" {
  count = 1

  triggers = {
    script_hash = filesha256("${path.module}/scripts/install-control-plane.sh")
  }

  depends_on = [openstack_networking_floatingip_associate_v2.tf_fip_assoc]

  connection {
    type        = "ssh"
    host        = openstack_networking_floatingip_v2.tf_fip[0].address
    user        = "ubuntu"
    private_key = file(var.key_pair_private_key_path)
    timeout     = "5m"
  }

  provisioner "file" {
    source      = "${path.module}/scripts/common.sh"
    destination = "/tmp/common.sh"
  }

  provisioner "file" {
    source      = "${path.module}/scripts/install-control-plane.sh"
    destination = "/tmp/install-control-plane.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/common.sh /tmp/install-control-plane.sh",
      "sudo /tmp/install-control-plane.sh ${var.k8s_minor_version}"
    ]
  }
}

resource "null_resource" "fetch_join_command" {
  count = var.node_count > 1 ? 1 : 0

  depends_on = [null_resource.k8s_control_plane]

  triggers = {
    always_run = timestamp()
  }

  provisioner "local-exec" {
    command = "mkdir -p ${path.module}/generated && ssh -o StrictHostKeyChecking=accept-new -i ${var.key_pair_private_key_path} ubuntu@${openstack_networking_floatingip_v2.tf_fip[0].address} 'sudo kubeadm token create --print-join-command' > ${path.module}/generated/join-command.sh"
  }
}

resource "null_resource" "k8s_worker" {
  count = var.node_count > 1 ? var.node_count - 1 : 0

  triggers = {
    script_hash = filesha256("${path.module}/scripts/install-worker.sh")
  }

  depends_on = [null_resource.fetch_join_command]

  connection {
    type        = "ssh"
    host        = openstack_networking_floatingip_v2.tf_fip[count.index + 1].address
    user        = "ubuntu"
    private_key = file(var.key_pair_private_key_path)
    timeout     = "5m"
  }

  provisioner "file" {
    source      = "${path.module}/scripts/common.sh"
    destination = "/tmp/common.sh"
  }

  provisioner "file" {
    source      = "${path.module}/scripts/install-worker.sh"
    destination = "/tmp/install-worker.sh"
  }

  provisioner "file" {
    source      = "${path.module}/generated/join-command.sh"
    destination = "/tmp/join-command.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/common.sh /tmp/install-worker.sh /tmp/join-command.sh",
      "sudo /tmp/install-worker.sh ${var.k8s_minor_version}"
    ]
  }
}
