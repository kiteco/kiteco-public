from datetime import datetime
from fabric.api import run
from fabric.api import task
from fabric.contrib import files
from fabric.operations import get, put
from fabric.state import env

# OpenVPN commands

_cmd_prefix = "sudo docker run -v /srv/openvpn:/etc/openvpn %s"
_azure_vpn_ip = "XXXXXXX"

@task
def initialize():
    host = "vpn.kite.com"
    if env.host == _azure_vpn_ip:
        host = "vpn-azure.kite.com"

    opts = "--rm XXXXXXX/openvpn ovpn_genconfig -u udp://%s:XXXXXXX" % host
    run(_cmd_prefix % opts)
    run(_cmd_prefix % "--rm -it XXXXXXX/openvpn ovpn_initpki")

@task
def start():
    run(_cmd_prefix % "-d -p XXXXXXX:XXXXXXX/udp --privileged XXXXXXX/openvpn")


def _get_id():
    return run("sudo docker ps | grep openvpn | awk '{print $1}'")

@task
def stop():
    container_id = _get_id()
    run("sudo docker stop %s" % container_id)

@task
def restart():
    container_id = _get_id()
    run("sudo docker restart %s" % container_id)

@task
def new_client(username):
    opts = "--rm -it XXXXXXX/openvpn easyrsa build-client-full %s" % username
    run(_cmd_prefix % opts)

@task
def revoke_client(username):
    opts = "--rm -it XXXXXXX/openvpn easyrsa revoke %s" % username
    run(_cmd_prefix % opts)

    opts = "--rm -it XXXXXXX/openvpn easyrsa gen-crl"
    run(_cmd_prefix % opts)

@task
def get_client_config(username):
    provider = "aws"
    if env.host == _azure_vpn_ip:
        provider = "azure"

    opts = "--rm XXXXXXX/openvpn ovpn_getclient %s > %s.ovpn" % (username, username)
    run(_cmd_prefix % opts)

    # Hack to make sure redirect-gateway line is gone.
    run("sed -n '/redirect-gateway/!p' %s.ovpn > %s-kite-vpn-%s.ovpn" % (username, username, provider))
    get(remote_path=("%s-kite-vpn-%s.ovpn" % (username, provider)))
    run("rm *.ovpn")

@task
def backup():
    provider = "aws"
    if env.host == _azure_vpn_ip:
        provider = "azure"

    ts = datetime.now().strftime("%Y-%m-%d-%H-%M-%S")
    run("sudo tar -cvzf openvpn-backup-%s-%s.tar.gz /srv/openvpn" % (provider, ts))
    get(remote_path="openvpn-backup-%s-%s.tar.gz" % (provider, ts), local_path="backup/")
    run("rm openvpn-backup-*.tar.gz")
