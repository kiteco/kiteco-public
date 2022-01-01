import os

from fabric.api import run
from fabric.api import task
from fabric.contrib import files
from fabric.context_managers import lcd, cd
from fabric.operations import put, local, sudo

import constants

@task
def init(hostname):
    install_puppet()
    update(hostname)

@task
def init_dev_instance():
    install_puppet()
    update("test.kite.com")

@task
def update(hostname):
    update_manifests(hostname)
    install_module()
    apply()

@task
def apply():
    sudo("puppet apply --modulepath=~/.puppet/modules default.pp")

@task
def install_puppet():
    sudo("apt-get -y update")
    sudo("apt-get -y install puppet")

@task
def install_module():
    with lcd(os.path.join(constants.KITE_ROOT, "devops", "puppet-kite")):
        local("puppet module build")
        put("pkg/kite-kite-*.tar.gz", "./")
        local("rm -rf pkg")

    run("rm -rf .puppet/modules")
    run("puppet module install --modulepath=~/.puppet/modules kite-kite-*.tar.gz")
    run("rm kite-kite-*.tar.gz")


@task
def update_manifests(hostname):
    if os.path.exists("hiera/%s.yaml" % hostname):
        sudo("mkdir -p /var/lib/hiera/")
        put("hiera/%s.yaml" % hostname, "/var/lib/hiera/common.yaml", use_sudo=True)
        sudo("mkdir -p /etc/puppetlabs/code/hieradata/")
        put("hiera/%s.yaml" % hostname, "/etc/puppetlabs/code/hieradata/common.yaml", use_sudo=True)
    else:
        print "No yaml file for host %s in hiera/" % hostname
        return
    if os.path.exists("manifests/%s.pp" % hostname):
        put("manifests/%s.pp" % hostname, "default.pp")
    else:
        print "No manifest for host %s in manifests/" % hostname
        return
