# TODO(naman) figure out how to either disable runtime hooks for other tests,
# or run pkgexploration tests in a separate process

import pytest
from . import runtime

# initialize runtime hooks at import time:
# as early as possible during pytest initialization
RUNTIME_REFMAP = runtime.patch()


@pytest.fixture(scope='session', autouse=True)
def runtime_refmap():
    return RUNTIME_REFMAP
