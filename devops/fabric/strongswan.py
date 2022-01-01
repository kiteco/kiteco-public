from datetime import datetime
from fabric.api import run
from fabric.api import task
from fabric.contrib import files
from fabric.operations import get, put

# strongswan commands


@task
def initialize():
    run("sudo apt-get update")
    run("sudo apt-get install -y strongswan")
    run("sudo sed -i 's/#net.ipv4.ip_forward/net.ipv4.ip_forward/g' /etc/sysctl.conf")
    run("sudo sysctl -p")

@task
def put_cfg(conf_file):
    put(remote_path="/etc/ipsec.conf", local_path=conf_file, use_sudo=True)


@task
def status():
    run("sudo ipsec status")


@task
def restart():
    run("sudo ipsec restart")
