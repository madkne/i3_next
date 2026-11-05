#cloud-config
# Minimal first-boot config. Everything else is done by guest-provision.sh
# so it can be re-run at any time (vm.sh provision).
hostname: __HOSTNAME__
manage_etc_hosts: true
users:
  - name: __USER__
    gecos: i3-next test user
    groups: [sudo, video, render, input, audio]
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    lock_passwd: false
    ssh_authorized_keys:
      - __SSH_PUBKEY__
# Throwaway console password (same as the user name) for the SPICE/serial console.
chpasswd:
  expire: false
  users:
    - {name: __USER__, password: __USER__, type: text}
ssh_pwauth: false
package_update: true
packages:
  - qemu-guest-agent
runcmd:
  - systemctl start qemu-guest-agent || true
