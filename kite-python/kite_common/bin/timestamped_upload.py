#!/usr/bin/env python
import time
import os
import boto
import argparse
from pprint import pprint
try:
    from urllib.parse import urlparse
except ImportError:
    from urlparse import urlparse

from kite.emr.constants import KITE_EMR_BUCKET


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--source', default="", required=True)
    parser.add_argument('--dest', default="", required=True)
    args = parser.parse_args()

    if args.dest.startswith("s3://"):
        url = urlparse(args.dest)
        dest_bucket = url.hostname
        dest_path = url.path[1:]
    else:
        dest_bucket = KITE_EMR_BUCKET
        dest_path = args.dest

    files = []
    if os.path.isfile(args.source):
        files.append((args.source, os.path.basename(args.source)))
    else:
        for dirpath, dirnames, filenames in os.walk(args.source):
            for fn in filenames:
                path = os.path.join(dirpath, fn)
                files.append((path, path[len(args.source)+1:]))

    ts = time.strftime('%Y-%m-%d_%H-%M-%S-%p')
    conn = boto.connect_s3()
    bucket = conn.get_bucket(dest_bucket)
    for fn, relpath in files:
        clean_fn = os.path.join(dest_path, ts, relpath)
        print("uploading to s3://%s/%s" % (dest_bucket, clean_fn))
        key = boto.s3.key.Key(bucket)
        key.key = clean_fn
        key.set_contents_from_filename(fn)

if __name__ == "__main__":
    main()
