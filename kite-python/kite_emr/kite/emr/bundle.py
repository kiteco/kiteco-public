import getpass
import os
import sys
import time
import subprocess
import glob
import shutil

import boto3

from kite.emr.constants import KITE_PYTHON
from kite.emr.constants import KITE_EMR_BUCKET
from kite.emr.constants import BUNDLE_DIR
from kite.emr.templates import template_with_root

try:
    from StringIO import StringIO  # python version < 3
except ImportError:
    from io import StringIO  # python version >= 3


class Path(object):
    def __init__(self, bucket, name=None, base=None):
        if name == None and base == None:
            raise Exception("must set job name or base directory")
        if name != None and base != None:
            raise Exception("must set only job name or base directory")

        self.bucket = bucket
        if name != None:
            ts = time.strftime('%Y-%m-%d_%H-%M-%S-%p')
            self.path = os.path.join(self._user_dir(), name, ts)
        if base != None:
            self.path = base

    def _user_dir(self):
        return os.path.join('users', getpass.getuser())

    def s3_path(self):
        return os.path.join("s3://", self.bucket, self.path)


class Bundle(object):
    def __init__(self, path):
        self.path = path
        self.bundle_dir = BUNDLE_DIR

    def _build_bootstrap_script(self):
        os.mkdir(os.path.join(self.bundle_dir, 'bootstrap'))
        fn = os.path.join(self.bundle_dir, 'bootstrap', 'bootstrap.sh')
        with open(fn, 'w') as fp:
            fp.write(template_with_root(self.path.s3_path()))

    def _build_kite_python(self):
        shutil.rmtree(os.path.join(KITE_PYTHON, "dist"))

        print("building kite-python module...")
        p = subprocess.Popen(['./setup.py', 'sdist'], cwd=KITE_PYTHON,
                         stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        p.wait()
        if p.returncode != 0:
            os.exit(1)

        matches = glob.glob(os.path.join(KITE_PYTHON, 'dist', 'kite-*.tar.gz'))
        if len(matches) == 1:
            shutil.copyfile(matches[0],
                            os.path.join(self.bundle_dir,
                                         'bootstrap', 'kite.tar.gz'))
            shutil.copyfile(os.path.join(KITE_PYTHON, 'requirements-emr.txt'),
                            os.path.join(self.bundle_dir,
                                         'bootstrap', 'requirements-emr.txt'))

    def _build_kite_go(self):
        env = os.environ.copy()
        env.update({
            'GOOS': 'linux',
            'GOARCH': 'amd64',
        })
        for dirpath, dirnames, filenames in os.walk(self.bundle_dir):
            if 'internal' in dirpath:
                continue
            for fn in filenames:
                if os.path.splitext(fn)[1] != ".go":
                    continue
                print("building %s" % os.path.join(dirpath, fn))
                p = subprocess.Popen(['go', 'build', fn], cwd=dirpath, env=env)
                p.wait()
                if p.returncode != 0:
                    sys.exit(p.returncode)

    def build(self):
        self.clean()
        if os.path.exists(self.bundle_dir):
            shutil.rmtree(self.bundle_dir)

        shutil.copytree(".", self.bundle_dir)
        self._build_bootstrap_script()
        self._build_kite_python()
        self._build_kite_go()

    def upload(self):
        files = []
        for dirpath, dirnames, filenames in os.walk(self.bundle_dir):
            for fn in filenames:
                files.append(os.path.join(dirpath, fn))

        s3 = boto3.resource('s3')
        for fn in files:
            clean_fn = os.path.join(self.path.path, fn[len(self.bundle_dir)+1:])
            try:
                file = open(fn, 'rb')
            except IOError as ex:
                sys.exit("error opening file %s: %s" % (fn, ex))
            s3.Object(self.path.bucket, clean_fn).put(Body=file)

    def clean(self):
        if os.path.exists(self.bundle_dir):
            shutil.rmtree(self.bundle_dir)
