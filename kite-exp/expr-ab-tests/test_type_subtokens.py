import logging
import os

logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(levelname)-8s %(message)s')

TEST_NAME = "test-type-subtokens"
PACKAGELIST = ["requests"]

KITECO_ROOT = os.path.join(os.getenv("HOME"), "go/src/github.com/kiteco/kiteco")
RUN_DIR = os.path.join(KITECO_ROOT, "local-pipelines/python-ggnn-expr-completion")
OUT = f"/data/expr-ab-tests/{TEST_NAME}/out"
TENSORBOARD = f"/data/expr-ab-tests/{TEST_NAME}/tensorboard"
RUNDB = f"s3://kite-data/run-db/users/damian/tests/{TEST_NAME}"


STEPS = 10000

os.makedirs(OUT, exist_ok=True)
os.makedirs(TENSORBOARD, exist_ok=True)


def run(cmd: str, in_dir=RUN_DIR):
    logging.info(f"running: {cmd}")

    cmd = f"set -e\ncd {in_dir}\n" + cmd

    ret = os.system(cmd)
    if ret != 0:
        raise RuntimeError("running '{}' returned non-zero status: {}".format(cmd, ret))


def write_file(filename: str, contents: str, rel_dir=RUN_DIR):
    logging.info(f"writing to {filename}")
    if not filename.startswith("/"):
        filename = os.path.join(rel_dir, filename)
    with open(filename, 'w+') as f:
        f.write(contents)


def start_graph_server(log_dir: str):
    logging.info("restarting graph server...")
    run(f"""
go run github.com/kiteco/kiteco/kite-go/lang/python/cmds/graph-data-server > {log_dir}/graph-data-server.log 2>&1 &
""")


def cleanup(experiment_name: str):
    logging.info("cleaning up...")
    run(f"""
pkill -f graph-data-server
make clean MODEL_NAME={experiment_name}
""")

def stop_graph_server():
    logging.info("stopping graph server...")
    try:
        run(f"""killall graph-data-server""")
    except Exception as exc:
        logging.info("no graph server running")


def wait_for_graph_server():
    logging.info("waiting for graph server...")
    run(f"""
while true; do
    if curl http://localhost:3039/some_bogus_page 2>&1 | grep -q '404 page not found'
    then
        break
    fi
    sleep 10
done
""")


def test_branch(branch: str, validate: bool = True):
    experiment_name = branch
    packagelist_file = f"tmp/packagelist.txt"
    out_dir = os.path.join(OUT, experiment_name)
    log_dir = os.path.join(OUT, experiment_name, "logs")
    tensorboard_dir = os.path.join(TENSORBOARD, experiment_name)

    logging.info(f"testing branch {branch}, steps={STEPS}")

    stop_graph_server()

    logging.info("setting up...")
    run(f"""
git checkout {branch}
mkdir -p tmp {log_dir}
""")
    write_file(packagelist_file, "\n".join(PACKAGELIST))

    start_graph_server(log_dir)
    wait_for_graph_server()

    logging.info("training...")
    run(f"""
make train \
    PACKAGES={packagelist_file} \
    MODEL_NAME={experiment_name} \
    OUT_DIR={out_dir} \
    TENSORBOARD_PATH={tensorboard_dir} \
    STEPS={STEPS} \
    > {log_dir}/train.log 2>&1""")

    if validate:
        logging.info("validating...")
        run(f"""
make validate_attribute MODEL_NAME={experiment_name} OUT_DIR={out_dir} > {log_dir}/attr-validate.log 2>&1
make validate_calls MODEL_NAME={experiment_name} OUT_DIR={out_dir} > {log_dir}/call-validate.log 2>&1
""")


def main():
    test_branch('master')
    test_branch('FEAT-type-subtokens')


if __name__ == "__main__":
    main()
