import argparse
import logging
import os


logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(levelname)-8s %(message)s')


def run(cmd: str):
    ret = os.system(cmd)
    if ret != 0:
        raise RuntimeError("running '{}' returned non-zero status: {}".format(cmd, ret))


def clean(host: str, directory: str):
    os.system("ssh {} 'killall graph_data_server'".format(host))
    run("ssh {} 'rm -rf {}/*'".format(host, directory))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--dir', type=str, help='directory to clean up on each remote machine')
    parser.add_argument('--hosts', nargs='*', type=str, help='list of hosts to deploy to')
    args = parser.parse_args()

    assert len(args.hosts) > 0, "need to clean at least one host"

    for host in args.hosts:
        logging.info("cleaning {} on {}".format(args.dir, host))
        clean(host, args.dir)


if __name__ == "__main__":
    main()
