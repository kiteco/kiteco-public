#!/usr/bin/env bash

set -e

# set up kite dir
sudo mkdir -p /var/kite
sudo mkdir -p /var/kite/log
sudo mkdir -p /var/kite/upload
sudo chown -R ubuntu.ubuntu /var/kite

cat << 'EOF' > /etc/sysctl.d/60-kite.conf
fs.file-max = 1048576
fs.nr_open = 1048576
net.nf_conntrack_max = 1048576
net.ipv4.tcp_mem = 786432 1697152 1945728
net.ipv4.tcp_rmem = 4096 4096 16777216
net.ipv4.tcp_wmem = 4096 4096 16777216
net.ipv4.ip_local_port_range = 1024 65535
net.netfilter.nf_conntrack_tcp_timeout_time_wait = 10
net.netfilter.nf_conntrack_tcp_timeout_established = 600
net.ipv4.tcp_slow_start_after_idle = 0
net.ipv4.tcp_tw_recycle = 0
net.ipv4.tcp_tw_reuse = 0
net.core.somaxconn = 65535
EOF

echo "Setting up disks"
sudo mkdir -p /data
if [[ -e /mnt ]]; then
    # instance has local storage
    sudo rmdir /data
    sudo mkdir /mnt/data
    sudo chown -R ubuntu.ubuntu /mnt/data
    sudo ln -s /mnt/data /data
elif [[ -e /dev/sdc ]]; then
    # instance has a disk mounted
    sudo mkfs -t ext4 /dev/sdc
    sudo mount /dev/sdc /data
fi
sudo mkdir -p /data/kite
sudo chown -R ubuntu.ubuntu /data/

# update the ssh config to automatically be able to ssh into other cluster instances without prompting
# note that the key itself (~/.ssh/kite-dev-azure) is uploaded via ssh, along with the bundle,
# as part of the deploy subcommand
# TODO: this is a little hacky, esp since the subnet is hardcoded
cat << 'EOF' >> /home/ubuntu/.ssh/config
Host 10.47.1.*
    User ubuntu
    IdentityFile ~/.ssh/kite-dev-azure
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
EOF

touch /var/kite/provisioned
