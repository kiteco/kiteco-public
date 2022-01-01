import os
import time
import datetime

from fabric.api import env, task, local, hosts, settings
from fabric.operations import run, put, get
from fabric.context_managers import cd, shell_env

env.user = 'ubuntu'
# env.key_filename = '~/.ssh/kite-dev.pem'
env.use_ssh_config = True

GOPATH=os.path.join("/mnt", "godeploy")
KITECO=os.path.join(GOPATH, "src/github.com/kiteco/kiteco")

ARTIFACTS=os.path.join("/mnt/kite", "artifacts")
RELEASES="/var/kite/releases"
LOGDIR="/var/kite/log"

BUILD_TARGETS = {
    "user-node": "github.com/kiteco/kiteco/kite-go/cmds/user-node",
    "user-mux": "github.com/kiteco/kiteco/kite-go/cmds/user-mux",
}

DOCKER_IMAGES = {}

DEPLOY_TARGETS = [
    "user-node",
    "user-mux",
]

@task
def create_release():
    branch = "release_%s" % _date()
    local("git checkout -b %s" % branch)
    local("git push -u origin %s" % branch)

@task
def create_client_release(client):
    branch = "release_%s_client_%s" % (_date(), client)
    local("git checkout -b %s" % branch)
    local("git push -u origin %s" % branch)


@task
@hosts('build.kite.com')
def build_release(branch):
    with shell_env(GOPATH=GOPATH, CGO_LDFLAGS_ALLOW='.*'):
        with cd(KITECO):
            # clear local changes if any
            run("git reset --hard")
            run("git checkout master")
            run("git pull")
            run("git checkout %s" % branch)
            run("git pull")

            run("make install-deps")

            artifacts_path = os.path.join(ARTIFACTS, branch)
            run("mkdir -p %s" % artifacts_path)

            for target, path in BUILD_TARGETS.items():
                run('go build -o %s %s' % (
                    os.path.join(artifacts_path, target), path))
                run("s3cmd put %s s3://kite-deploys/%s/%s" % (
                    os.path.join(artifacts_path, target),
                    branch,
                    target))

            for tar, folder in DOCKER_IMAGES.items():
                with cd(folder):
                    run("make save OUTPUT=%s/%s" % (artifacts_path, tar))
                    run("s3cmd put %s s3://kite-deploys/%s/%s" % (
                        os.path.join(artifacts_path, tar),
                        branch,
                        tar))

@task
@hosts('build.kite.com')
def pull_release(branch):
    local("mkdir -p %s" % branch)
    artifacts_path = os.path.join(ARTIFACTS, branch)
    for target in DEPLOY_TARGETS:
        get(remote_path=os.path.join(artifacts_path, target),
            local_path=branch)

@task
def push_release(branch):
    run("mkdir -p %s" % os.path.join(RELEASES, branch))
    for target in DEPLOY_TARGETS:
        put(local_path=os.path.join(branch, target),
            remote_path=os.path.join(RELEASES, branch),
            mode=0o755)
    local("rm -rf %s" % branch)

@task
def deploy_release(branch):
    current_dir = os.path.join(RELEASES, "current")
    branch_dir = os.path.join(RELEASES, branch)

    run("rm -f %s" % current_dir)
    run("ln -s %s %s" % (branch_dir, current_dir))

    for target in DEPLOY_TARGETS:
        executable_file = os.path.join(branch_dir, target)
        log_file = os.path.join(LOGDIR, "%s.log" % target)

        with settings(warn_only=True):
            run("killall %s" % target)
            time.sleep(5)
        run("nohup %s &> %s &" % (executable_file, log_file), pty=False)


# Release server tasks --------------------------------------------------

@task
def build_and_deploy_release_server():
    branch = create_release_release()
    build_release_server(branch)
    deploy_release_server(branch)

@task
def create_release_release():
    branch = "release_server_%s" % _date()
    local('git checkout -b %s' % branch)
    local('git push -u origin %s' % branch)
    return branch

@task
@hosts('build.kite.com')
def build_release_server(branch):
    with shell_env(GOPATH=GOPATH, CGO_LDFLAGS_ALLOW='.*'):
        with cd(KITECO):
            run("git fetch")
            run("git checkout %s" % branch)

            run("make install-deps")

            target = 'release'
            go_pkg_path = 'github.com/kiteco/kiteco/kite-go/cmds/release'

            artifacts_dir = os.path.join(ARTIFACTS, branch)
            artifacts_path = os.path.join(artifacts_dir, target)

            run("mkdir -p %s" % artifacts_dir)
            run('go build -o %s %s' % (artifacts_path, go_pkg_path))
            run("s3cmd put %s s3://kite-deploys/%s/%s" % (artifacts_path, branch, target))

@task
def deploy_release_server(branch):
    current_dir = os.path.join(RELEASES, "current")
    branch_dir = os.path.join(RELEASES, branch)

    run('rm -rf %s' % branch_dir)
    run('mkdir %s' % branch_dir)
    run('s3cmd get s3://kite-deploys/%s/release %s' % (branch, branch_dir))

    run('rm -f %s' % current_dir)
    run('ln -s %s %s' % (branch_dir, current_dir))

    target = 'release'
    executable_file = os.path.join(branch_dir, target)
    run('chmod 755 %s' % executable_file)
    log_file = os.path.join(LOGDIR, "%s.log" % target)

    with settings(warn_only=True):
        run('killall %s' % target)
        time.sleep(5)
    run('nohup %s server &> %s &' % (executable_file, log_file), pty=False)

@task
def deploy_mock_release_server(branch):
    current_dir = os.path.join(RELEASES, "current")
    branch_dir = os.path.join(RELEASES, branch)

    run('rm -rf %s' % branch_dir)
    run('mkdir %s' % branch_dir)
    run('s3cmd get s3://kite-deploys/%s/release %s' % (branch, branch_dir))

    run('rm -f %s' % current_dir)
    run('ln -s %s %s' % (branch_dir, current_dir))

    target = 'release'
    executable_file = os.path.join(branch_dir, target)
    run('chmod 755 %s' % executable_file)
    log_file = os.path.join(LOGDIR, "%s.log" % target)

    with settings(warn_only=True):
        run('killall %s' % target)
        time.sleep(5)

    run('nohup %s mockserver &> %s &' % (executable_file, log_file), pty=False)

## --

def _date():
    return datetime.datetime.now().strftime("%Y%m%dT%H%M%S")
