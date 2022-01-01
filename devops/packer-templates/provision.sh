#!/bin/bash

cat << 'EOF' > /home/ubuntu/60-kite.conf
fs.file-max = 10000000
fs.nr_open = 10000000
net.ipv4.tcp_mem = 786432 1697152 1945728
net.ipv4.tcp_rmem = 4096 4096 16777216
net.ipv4.tcp_wmem = 4096 4096 16777216
net.ipv4.ip_local_port_range = 2000 65535
net.ipv4.netfilter.ip_conntrack_tcp_timeout_time_wait = 10
net.netfilter.nf_conntrack_tcp_timeout_established = 600
net.ipv4.tcp_slow_start_after_idle = 0
net.ipv4.tcp_tw_recycle = 0
net.ipv4.tcp_tw_reuse = 0
net.core.somaxconn = 65535
EOF
sudo cp /home/ubuntu/60-kite.conf /etc/sysctl.d/60-kite.conf
rm /home/ubuntu/60-kite.conf

cat << 'EOF' > /home/ubuntu/kite.conf
* soft nofile 10000000
* hard nofile 10000000
root soft nofile 10000000
root hard nofile 10000000
EOF

sudo cp /home/ubuntu/kite.conf /etc/security/limits.d/kite.conf
rm /home/ubuntu/kite.conf
