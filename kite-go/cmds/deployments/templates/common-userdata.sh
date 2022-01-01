#!/bin/bash

set -e

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

# Format and setup the instance store
if ! `findmnt -rno SOURCE /dev/xvdb > /dev/null`; then
    if [[ -e /dev/xvdb ]]; then
        mkfs -t ext4 /dev/xvdb
        mount -a
        mkdir /mnt/kite
    fi
fi

# Get pagerduty..
# wget -O - http://packages.pagerduty.com/GPG-KEY-pagerduty | apt-key add -
# echo "deb http://packages.pagerduty.com/pdagent deb/" >/etc/apt/sources.list.d/pdagent.list

# Install some things
apt-get -y update
apt-get -y install htop python-pip # pdagent pdagent-integrations
pip install s3cmd

# Install libtensorflow
curl -L "https://s3-us-west-1.amazonaws.com/kite-data/tensorflow/libtensorflow-cpu-linux-x86_64-1.15.0.tar.gz" | sudo tar -C /usr/local -xz
sudo ldconfig

# Setup directories
ln -s /mnt/kite /var/kite
mkdir -p /mnt/kite/log /mnt/kite/releases /mnt/kite/s3cache /mnt/kite/certs /mnt/kite/tmp

# Get RDS certificates
s3cmd get s3://rds-downloads/rds-combined-ca-bundle.pem /var/kite/certs/rds-combined-ca-bundle.pem

# Get binary
RELEASE_DIR=/var/kite/releases/{{.ReleaseBin}}
mkdir -p $RELEASE_DIR
s3cmd get s3://kite-deploys/{{.ReleaseBin}}/{{.Process}} $RELEASE_DIR/{{.Process}}
chmod +x $RELEASE_DIR/{{.Process}}

# Get configuration
CONFIG=s3://XXXXXXX/config/{{.Region}}.sh
s3cmd get $CONFIG /var/kite/config.sh
chmod +x /var/kite/config.sh

# Permissions
chown -R ubuntu:ubuntu /mnt/kite
chown -R ubuntu:ubuntu /var/kite

source /var/kite/config.sh

cat << 'EOF' > /etc/td-agent/config.d/500-serverlogs.conf
<source>
  @type tail
  path /var/kite/log/user-mux.log
  pos_file /var/log/td-agent/user-mux.log.pos
  tag user-mux.log
  <parse>
    @type none
  </parse>
</source>

<source>
  @type tail
  path /var/kite/log/user-node.log
  pos_file /var/log/td-agent/user-node.log.pos
  tag user-node.log
  <parse>
    @type none
  </parse>
</source>

<filter user-node.log>
  @type grep
  <exclude>
    key message

    # "session logged out": Community error when user is logged out
    # "empty token data": Request does not include token data
    # "session": Session db lookups part 1
    # "users.go:333": Session db lookups part 2
    # "value too long for type character varying": Unclear where this is coming from
    # "[GET|POST]+ [\S]+ [2|0]": Any endpoint that responds with 2xx (or 0)
    # "/api/account/authenticated": Ignore response from authenticated endpoint
    # "fetching s3": Fetching local index data from s3
    # "/api/buffer": Buffer endpoint, not caught by other http regex since it contains whitespace
    # "error adding file: parse error": Python builder parse error
    # "found multiple matching distributions for path": pythonresource.PathSymbol error
    # "The specified key does not exist": S3 Content store Get error
    # "error getting file:": Builder Get error
    # "GET /api/account/user 401": Reports user logged out
    # "GET /api/local_code_status 500": Local code status
    # "backslash not followed by newline": Parsing error
    # "POST /user/status 404": Status upload
    # "illegal character": Parsing error
    # "string literal not terminated": Parsing error
    # "illegal Utf-8 encoding": Parsing error
    # "users.go:312": Community db query
    # "got unexpected type:": Editor services
    # "GET /api/local_code_status 404": Local code status issue
    # "errors.go:108: app error": Community errors
    # "error cleaning up cache in putWriter:": Missing file in diskcache
    # "status code: 404, request id:": Prefetch get content failures
    pattern /session logged out|empty token data|"session"|users\.go:333|value too long for type character varying|[GET|POST]+ [\S]+ [2|0]|\/api\/account\/authenticated|fetching s3|\/api\/buffer|error adding file: parse error|found multiple matching distributions for path|The specified key does not exist|error getting file:|GET \/api\/account\/user 401|GET \/api\/local_code_status 500|backslash not followed by newline|POST \/user\/status 404|illegal character|string literal not terminated|illegal Utf-8 encoding|users\.go:312|got unexpected type:|GET \/api\/local_code_status 404|errors\.go:108: app error|error cleaning up cache in putWriter:|status code: 404, request id:/
  </exclude>
</filter>

<filter **>
  @type ec2_metadata

  <record>
    instance_id   ${instance_id}
    instance_type ${instance_type}
    az            ${availability_zone}
    private_ip    ${private_ip}
  </record>
</filter>

<match user-node.log>
  @type kinesis_firehose
  delivery_stream_name server-logs
  region us-east-1

  <inject>
    time_key timestamp
    time_type string
    time_format %Y-%m-%dT%H:%M:%SZ
  </inject>
</match>
EOF
service td-agent restart

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
export REGION={{.Region}}
export RELEASE={{.Release}}
export PROVIDER="aws"
export AWS_REGION={{.Region}}
export LIBRATO_SOURCE={{.Region}}.{{.Process}}.{{.ReleaseNoDots}}.$HOSTNAME
export LOCAL_WORKER_TAG={{.Release}}
export LOCAL_WORKER_REQUEST_QUEUE=local-analysis-request-{{.ReleaseMD5}}
export LOCAL_WORKER_RESPONSE_QUEUE=local-analysis-response-{{.ReleaseMD5}}

# Dump limits/sysctl
ulimit -a > /var/kite/log/ulimit.log
sysctl -a > /var/kite/log/sysctl.log

export LD_LIBRARY_PATH=/usr/local/lib

# Go 1.12 by default uses GODEBUG=MADVFREE which may report higher RSS.
# See https://golang.org/doc/go1.12#runtime
export GODEBUG="madvdontneed=1"

# Run forever. Report to pager duty if node was restarted
until `/var/kite/releases/{{.ReleaseBin}}/{{.Process}} &>> /var/kite/log/{{.Process}}.log`; do
    echo "exited with return code $?; restarting {{.Process}} ..."

    # Timestamp and save the current log
    DATE=`date +"%m-%d-%Y-%T"`
    mv /var/kite/log/{{.Process}}.log /var/kite/log/{{.Process}}.$DATE.log

    # Write out the stack trace from the crash into a separate log - we do this by using tac to read the log in reverse, finding and printing up to the last go logger output using sed, then reversing it again
    tac /var/kite/log/{{.Process}}.$DATE.log | sed -n '1,/\[region=/p' | tac > /var/kite/log/crash-{{.Process}}.$DATE.log

    # Send to pagerduty
    {{ if .IsProduction }}
    # pd-send \
    #   -k $PG_SERVICE_KEY \
    #   -t trigger \
    #   -d "$LIBRATO_SOURCE at $HOSTNAME died, restarting" \
    #   -i "{{.Region}}.{{.Process}}.{{.ReleaseNoDots}}.died_restarting" \
    #   -f "region={{.Region}}" \
    #   -f "release={{.Release}}" \
    #   -f "process={{.Process}}" \
    #   -f "hostname=$HOSTNAME" \
    #   -f "logs=$(head -c 100000 /var/kite/log/crash-{{.Process}}.$DATE.log)"
    {{ end }}
    sleep 10
done
EOF

chmod +x run.sh
nohup ./run.sh &> /var/kite/log/watchdog.log &
