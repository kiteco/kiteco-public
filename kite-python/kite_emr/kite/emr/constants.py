import os

PIPELINE_FILE = 'pipeline.yaml'
# KITE_COMMON_ROOT is the path to kite-python/kite_emr in the kiteco repo
KITE_PYTHON = os.environ.get("KITE_EMR_ROOT",
                             os.path.join(os.environ["GOPATH"], "src/github.com/kiteco/kiteco/kite-python/kite_emr"))
KITE_EMR_BUCKET = 'kite-emr'
BUNDLE_DIR = 'bundle'
