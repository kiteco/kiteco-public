#!/bin/bash

set -e

echo "Configuring for region {{.Location}} release {{.Release}}"

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

# Setup directories in /dev/sdb (which is automatically mounted)
mkdir /mnt/kite
ln -s /mnt/kite /var/kite
chown -R ubuntu:ubuntu /mnt/kite
chown -R ubuntu:ubuntu /var/kite
mkdir -p /mnt/kite/log /mnt/kite/releases /mnt/kite/s3cache /mnt/kite/certs /mnt/kite/tmp

# Install dependencies

# deal with 14.04 specific apt-get issue
sudo apt-get clean
sudo mv /var/lib/apt/lists /tmp
sudo mkdir -p /var/lib/apt/lists/partial
sudo apt-get clean

# pagerduty
wget -O - http://packages.pagerduty.com/GPG-KEY-pagerduty | apt-key add -
echo "deb http://packages.pagerduty.com/pdagent deb/" >/etc/apt/sources.list.d/pdagent.list

# sometimes the vms don't get all the ubuntu primary repositories when provisioning, so we manually
# add them here to make sure the apt-get works
sudo add-apt-repository main
sudo add-apt-repository universe
sudo add-apt-repository restricted
sudo add-apt-repository multiverse

sudo apt-get -y update

# sometimes even with the add-apt-repository, apt-get is unable to find the packages, so we just
# keep trying until it does
until sudo apt-get -y install htop pdagent pdagent-integrations python-pip
do
  sudo add-apt-repository main
  sudo add-apt-repository universe
  sudo add-apt-repository restricted
  sudo add-apt-repository multiverse
  sudo apt-get -y update
done

# Install libtensorflow
curl -L "https://s3-us-west-1.amazonaws.com/kite-data/tensorflow/libtensorflow-cpu-linux-x86_64-1.15.0.tar.gz" | sudo tar -C /usr/local -xz
sudo ldconfig

# pip
pip install s3cmd

# Set AWS credentials for s3cmd
export AWS_ACCESS_KEY_ID="{{.AWSID}}"
export AWS_SECRET_ACCESS_KEY="{{.AWSKey}}"
export AWS_REGION="{{.AWSRegion}}"

# Get credentials from AWS
CONFIG=s3://XXXXXXX/config/azure-{{.Location}}.sh
s3cmd get $CONFIG /var/kite/config.sh
chmod +x /var/kite/config.sh
source /var/kite/config.sh

# Get RDS certificates
s3cmd get s3://rds-downloads/rds-combined-ca-bundle.pem /var/kite/certs/rds-combined-ca-bundle.pem

# Get binary
RELEASE_DIR=/var/kite/releases/{{.Release}}
mkdir -p $RELEASE_DIR
s3cmd get s3://kite-deploys/{{.Release}}/{{.Process}} $RELEASE_DIR/{{.Process}}
chmod +x $RELEASE_DIR/{{.Process}}

# Setup logging to papertrail
cat << 'EOF' > /etc/log_files.yml
files:
  - /var/log/dmesg
  - /var/log/syslog
  - /var/kite/log/user-node.log
  - /var/kite/log/user-mux.log
destination:
  host: PAPERTRAIL_HOST
  port: PAPERTRAIL_PORT
  protocol: tls
EOF
sed -i -e "s|PAPERTRAIL_HOST|$PAPERTRAIL_HOST|" /etc/log_files.yml
sed -i -e "s|PAPERTRAIL_PORT|$PAPERTRAIL_PORT|" /etc/log_files.yml

# Get remote_syslog2 binary for watching logs
UTILS_DIR=/var/kite/utils
mkdir -p $UTILS_DIR
s3cmd get s3://kite-utils/remote_syslog2/remote_syslog_linux_amd64.tar.gz $UTILS_DIR
cd $UTILS_DIR
tar xzf remote_syslog*.tar.gz
cd remote_syslog
cp remote_syslog /usr/local/bin

# Apply sysctl to current session
sysctl --system

# Update max open files for current session
ulimit -n 1048576

# Add logrotate config
cat << 'EOF' | sudo tee -a /etc/logrotate.conf

/var/kite/log/{{.Process}}.log {
    missingok
    size 1G
    rotate 5
    compress
    copytruncate # this is needed for usernode to continue writing to the same file
}
EOF

# Add crontab for logrotate
echo "0 * * * * sudo logrotate /etc/logrotate.conf" > /var/kite/mycron
crontab /var/kite/mycron

cat << 'EOF' > run.sh
#!/bin/bash

# Source configuraiton
source /var/kite/config.sh

# Release specific configuration
export HOSTNAME=$(hostname)
export REGION={{.Location}}
export RELEASE={{.Release}}
export PROVIDER="azure"
export LIBRATO_SOURCE={{.Location}}.{{.Process}}.{{.ReleaseNoDots}}.$HOSTNAME
export LOCAL_WORKER_TAG={{.Release}}
export LOCAL_WORKER_REQUEST_QUEUE=local-analysis-request-{{.ReleaseMD5}}
export LOCAL_WORKER_RESPONSE_QUEUE=local-analysis-response-{{.ReleaseMD5}}
export AWS_REGION={{.AWSRegion}}
export AZURE_REGION={{.Location}}

# Start remote_syslog
remote_syslog --configfile=/etc/log_files.yml --debug-log-cfg=/var/log/remote_syslog.log

# Dump limits/sysctl
ulimit -a > /var/kite/log/ulimit.log
sysctl -a > /var/kite/log/sysctl.log

# Run forever. Report to pager duty if node was restarted
until `/var/kite/releases/{{.Release}}/{{.Process}} &>> /var/kite/log/{{.Process}}.log`; do
    echo "exited with return code $?; restarting {{.Process}} ..."

    # Timestamp and save the current log
    DATE=`date +"%m-%d-%Y-%T"`
    mv /var/kite/log/{{.Process}}.log /var/kite/log/{{.Process}}.$DATE.log

    # Write out the stack trace from the crash into a separate log - we do this by using tac to read the log in reverse, finding and printing up to the last go logger output using sed, then reversing it again
    tac /var/kite/log/{{.Process}}.$DATE.log | sed -n '1,/\[region=/p' | tac > /var/kite/log/crash-{{.Process}}.$DATE.log

    # Send to pagerduty
    {{ if .IsProduction }}
    pd-send \
      -k $PG_SERVICE_KEY \
      -t trigger \
      -d "$LIBRATO_SOURCE at $HOSTNAME died, restarting" \
      -i "{{.Location}}.{{.Process}}.{{.ReleaseNoDots}}.died_restarting" \
      -f "region={{.Location}}" \
      -f "release={{.Release}}" \
      -f "process={{.Process}}" \
      -f "hostname=$HOSTNAME" \
      -f "logs=$(head -c 100000 /var/kite/log/crash-{{.Process}}.$DATE.log)"
    {{ end }}
    sleep 10
done
EOF

chmod +x run.sh
nohup ./run.sh &> /var/kite/log/watchdog.log &
