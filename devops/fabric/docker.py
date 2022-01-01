from fabric.api import run
from fabric.api import task
from fabric.contrib import files

# Install docker on the machine ---

def has_docker():
    return files.exists("/usr/local/bin/docker")

def install_docker():
    run("sudo apt-get -y update")
    run("sudo apt-get -y install docker.io")
    run("sudo ln -sf /usr/bin/docker.io /usr/local/bin/docker")

@task
def provision():
    if not has_docker():
        install_docker()

@task
def ps():
    run("sudo docker ps")

@task
def stop(container_id):
    run("sudo docker stop %s" % container_id)

@task
def restart(container_id):
    run("sudo docker restart %s" % container_id)
