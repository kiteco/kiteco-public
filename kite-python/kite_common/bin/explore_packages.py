#!/usr/bin/env python

"""
This script imports a set of packages, then explores them by recursively calling dir() on all modules
and types, and finally outputs a json representation of everything it found.

Usage:

    python explore_packages.py LABEL PACKAGE PACKAGE ... --graph_output=graph.json --dependency_output=deps.txt

To run exploration on the sys package:

    python explore_packages.py sys sys ... --graph_output=graph.json --dependency_output=deps.txt

To run exploration on several packages:

    python explore_packages.py scientific numpy scipy pandas ... --graph_output=graph.json --dependency_output=deps.txt
"""
from kite.pkgexploration.runtime import patch; REFMAP = patch()

import argparse
import gc
import json
import logging
import sys
import warnings

from kite.pkgexploration.explore import Explorer

logger = logging.getLogger()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("package", nargs=1, help="names of package to import")
    parser.add_argument("--graph", required=True, help="output for the import graph (json)")
    parser.add_argument("--dependencies", help="output for package dependency information (text)")
    parser.add_argument("--verbose", default=False, action="store_true", help="verbose output mode")
    args = parser.parse_args()

    # Importing everything in the whole python universe tends to generate a large number of unhelpful warnings
    warnings.filterwarnings("ignore")

    if args.verbose:
        logging.basicConfig(level=logging.DEBUG)
    else:
        logging.basicConfig(level=logging.INFO)

    # It turns out that importing certain files actually causes changes to the __builtin__ package
    # so keep a reliable reference to the stuff that we'll need later when we're in sparta
    realopen = open
    reallen = len
    realsys = sys
    realjson = json

    # Disable garbage collection since otherwise it is difficult to keep IDs consistent
    gc.disable()

    # Run the traversal
    explorer = Explorer.explore_package(args.package[0], refmap=REFMAP)
    if explorer is None:
        logger.error("exploration failed")
        return

    # Dump the results
    with realopen(args.graph, "w") as f:
        for info in explorer.info_by_id.values():
            try:
                # Do _not_ do json.dump(info, f) here because if there is a serialization error then we
                # will have malformed output.
                f.write(realjson.dumps(info))
                f.write('\n')
            except UnicodeDecodeError as e:
                logger.error("Unable to dump info for %s due to unicode error" % info["canonical_name"])

    # Look at which packages were imported as a result of this exploration
    if args.dependencies:
        with realopen(args.dependencies, "w") as f:
            for pkg in realsys.modules:
                f.write(pkg + "\n")

    logger.info("Wrote %d items to %s" % (reallen(explorer.info_by_id), args.graph))
    logger.info("Wrote %d dependencies to %s" % (reallen(realsys.modules), args.dependencies))


if __name__ == "__main__":
    main()
