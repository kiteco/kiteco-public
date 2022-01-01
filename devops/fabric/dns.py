from fabric.api import run
from fabric.api import task
from fabric.contrib import files
from fabric.operations import get, put

@task
def backup():
    get(remote_path="/etc/dnsmasq.conf", local_path="backup/dns/dnsmasq.conf")
    get(remote_path="/etc/hosts", local_path="backup/dns/hosts")

