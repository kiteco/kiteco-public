from typing import Dict

import argparse
import logging
import math
import os

logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(levelname)-8s %(message)s')

# TODO: Perhaps use fabric?


def run(cmd: str):
    ret = os.system(cmd)
    if ret != 0:
        raise RuntimeError("running '{}' returned non-zero status: {}".format(cmd, ret))


def make_bundle(bundle: str, meta_info: str, kite_ml_path: str, env_vars: Dict[str, str]):
    run("rm -rf {} && mkdir {}".format(bundle, bundle))

    with open("{}/env.sh".format(bundle), 'w') as f:
        for k, v in env_vars.items():
            f.write("{}={}\n".format(k, v))

    run("go build -o {}/graph_data_server github.com/kiteco/kiteco/kite-go/lang/python/cmds/graph-data-server".format(
        bundle
    ))
    run("cp -v {} {}/metainfo.json".format(meta_info, bundle))
    run("mkdir {}/kite_ml".format(bundle))
    run("cp -v {}/requirements.txt {}/kite_ml".format(kite_ml_path, bundle))
    run("cp -rv {}/kite {}/kite_ml".format(kite_ml_path, bundle))
    run("cp -v datagen/get_data.py {}".format(bundle))
    run("cp -v datagen/run.sh {}".format(bundle))
    run("cp -v datagen/start.sh {}".format(bundle))

    bundle_file = "{}.tar.gz".format(bundle)
    run("tar czvf {} {}".format(bundle_file, bundle))


def deploy_to_host(bundle: str, host: str, random_seed: int):
    run("scp {}.tar.gz {}:.".format(bundle, host))
    run("ssh {} 'rm -rf {} && tar xzf {}.tar.gz && cd {} && RANDOM_SEED={} ./start.sh'".format(
        host, bundle, bundle, bundle, random_seed))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--bundle', type=str, help='bundle name')
    parser.add_argument('--steps', type=int, help='total number of training steps we expect to run')
    parser.add_argument('--batch', type=int, default=10, help='batch size')
    parser.add_argument('--meta_info', type=str, help='path to meta_info')
    parser.add_argument('--max_samples', type=int)
    parser.add_argument('--attr_base_proportion', type=float)
    parser.add_argument('--attr_proportion', type=float)
    parser.add_argument('--call_proportion', type=float)
    parser.add_argument('--arg_type_proportion', type=float)
    parser.add_argument('--kwarg_name_proportion', type=float)
    parser.add_argument('--arg_placeholder_proportion', type=float)
    parser.add_argument('--samples_per_file', type=int, default=500)
    parser.add_argument('--kite_ml_path', type=str, help='path to Kite ML')
    parser.add_argument('--out_dir', type=str, help='output directory that each instance will write to locally')
    parser.add_argument('--hosts', nargs='*', type=str, help='list of hosts to deploy to')
    args = parser.parse_args()

    assert len(args.hosts) > 0, "need to deploy to at least one host"

    logging.info("deploying to {} hosts:".format(len(args.hosts)))
    for host in args.hosts:
        logging.info("* {}".format(host))

    # continue to generate samples until we are killed to account for some
    # instances producing samples at different rates
    samples_per_host = args.steps
    env_vars = {
        'SAMPLES': samples_per_host,
        'OUT_DIR': args.out_dir,
        'BATCH': args.batch,
        'MAX_SAMPLES': args.max_samples,
        'ATTR_PROPORTION': args.attr_proportion,
        'ATTR_BASE_PROPORTION': args.attr_base_proportion,
        'CALL_PROPORTION': args.call_proportion,
        'ARG_TYPE_PROPORTION': args.arg_type_proportion,
        'KWARG_NAME_PROPORTION': args.kwarg_name_proportion,
        'ARG_PLACEHOLDER_PROPORTION': args.arg_placeholder_proportion,
        'SAMPLES_PER_FILE': args.samples_per_file,
        'KITE_AZUREUTIL_STORAGE_KEY': 'XXXXXXX',
        'KITE_AZUREUTIL_STORAGE_NAME': 'kites3mirror',
        'KITE_USE_AZURE_MIRROR': '1',
    }

    make_bundle(args.bundle, args.meta_info, args.kite_ml_path, env_vars)

    for i, host in enumerate(args.hosts):
        logging.info("deploying {}.tar.gz to {}".format(args.bundle, host))
        random_seed = i + 1
        deploy_to_host(args.bundle, host, random_seed)


if __name__ == "__main__":
    main()
