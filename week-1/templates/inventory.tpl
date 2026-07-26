[control_plane]
%{ for idx, ip in floating_ips ~}
%{ if idx == 0 ~}
tf-vm-${idx + 1} ansible_host=${ip}
%{ endif ~}
%{ endfor ~}

[workers]
%{ for idx, ip in floating_ips ~}
%{ if idx > 0 ~}
tf-vm-${idx + 1} ansible_host=${ip}
%{ endif ~}
%{ endfor ~}

[all:children]
control_plane
workers
