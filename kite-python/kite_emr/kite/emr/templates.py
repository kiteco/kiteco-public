import os

_BOOTSTRAP_TEMPLATE = """#!/bin/bash
aws s3 cp $S3ROOT/bootstrap/requirements-emr.txt ./
aws s3 cp $S3ROOT/bootstrap/kite.tar.gz ./
sudo alternatives --set python /usr/bin/python2.7
sudo yum-config-manager --enable epel
sudo pip install -r requirements-emr.txt
sudo pip install kite.tar.gz

sudo mkdir -p /var/kite
sudo chown -R hadoop:hadoop /var/kite
mkdir -p /var/kite/s3cache/tmp
"""


def template_with_root(root):
    return (_BOOTSTRAP_TEMPLATE.replace("$S3ROOT", root))
