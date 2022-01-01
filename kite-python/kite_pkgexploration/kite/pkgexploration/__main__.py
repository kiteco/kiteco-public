from kite.pkgexploration.runtime import patch; REFMAP = patch()

import argparse
import gc
import json
import logging
import os
import pkg_resources
import sys
import warnings

from kite.pkgexploration.explore import Explorer
from kite.pkgexploration.dist import DistMeta

_PY3 = sys.version >= '3'
logger = logging.getLogger(__name__)

# Ignore packages that start with any of these strings
BLACKLIST_PREFIX = [
    "kite",
    "__main__",
    "antigravity",
    "IPython",
    "appletrunner",
    "macropy.console",
    "pyinotify",        # because it cause explore_packages to segfault
    "keyring",          # because it requires user input
    "keystoneclient",   # somehow causes the whole import loop to abort!
    "glanceclient",     # requires user input
    "novaclient",       # requires user input
    "twisted",          # because it hangs
    "bsddb",
    "ctypes.wintypes",
    "lib2to3",
    "tests",            # some packages expose these as a top level name and they always fail
    "test",             # python builtin tests
    "gevent.tests",     # gevent.tests has terrible side-effects
    "pandas._libs",     # get_argspec segfaults on some C funcs
    "kivy.tools",       # segfaults, and is just cli tools, examples, etc
]

# Ignore packages that contain any of these strings
BLACKLIST_PATTERN = [
    "pygame.examples",
    "pygame.tests",
    "cherrypy.test"
    "pandas.util.clipboard",
    "twisted.test",  # long running tests
    "keystonemiddleware.echo",  # starts a server when imported
    "statsmodels.tests",  # long running tests
    "superlance.grower",  # infinite loop that leaks memory
    "zdaemon.tests",  # long running tests, some infinite loops
    "jpype",  # causes a seg fault
    "sympy",  # sometimes causes a seg fault
    "cp65001",  # only on windows
    "venv",  # requires a virtual env
    "asyncio.windows_util",  # windows only
    "asyncio.windows_events",  # windows only
    "dbm.gnu",  # interface to unix databases, fails to import
    "distutils._msvccompiler",  # windows only
    "distutils.command.bdist_msi",  # windows only
    "distutils.msvc9compiler",  # windows only
    "distutils._msvccompiler",  # windows only
    "distutils.command.bdist_msi",  # windows only
    "encodings.mbcs",  # windows only
    "multiprocessing.popen_spawn_win32",  # windows only
    "jenkinsapi_utils.simple_post_logger",  # hangs on import
    "flake8.__main__",  # hangs on import
    "django.contrib.gis",  # requires C library GDAL
    "django.db.backends.oracle",  # requires cx_Oracle; what Python developer uses Oracle?
    "marionette_client",  # causes an ImportError
    "google.protobuf",  # segfaults
    "jpype",  # causes a seg fault
    "sklearn.linear_model.cd_fast",  # causes seg fault
    "sklearn.cluster.k_means_",  # causes seg fault
    "sklearn.cluster._k_means_elkan",  # causes seg fault
    "kivy.loader", # causes seg fault
]

# Ignore these when processing the std lib
BLACKLIST_PATTERN_STDLIB = [
    "pip",
    "six",
    "setuptools",
]


def _check_prefix(s, pfx):
    return s.startswith(pfx) and (len(s) == len(pfx) or s[len(pfx)] == ".")


def _check_pattern(s, pat):
    if len(s) < len(pat):
        return False

    return (_check_prefix(s, pat) or
            (s.endswith(pat) and s[-len(pat) - 1] == ".") or
            (".{}.".format(pat) in s))


def blacklisted(name):
    return any(_check_prefix(name, prefix) for prefix in BLACKLIST_PREFIX) or \
        any(_check_pattern(name, pattern) for pattern in BLACKLIST_PATTERN)


def blacklisted_stdlib(name):
    return blacklisted(name) or any(_check_pattern(name, pattern) for pattern in BLACKLIST_PATTERN_STDLIB)


def _cleanup_names(names):
    cleaned = set()
    for name in names:
        name = name.split(".", 1)[0].strip()
        if len(name) > 0:
            cleaned.add(name)
    return cleaned


def top_level_names(metadata, from_user=None):
    """Try to find the top level import names for a pkg and version"""
    names = set()

    if from_user:
        names.update(from_user)

    if metadata:
        names.update(metadata.top_level_names)

    return _cleanup_names(names)


def stdlib_names():
    # we only want global modules, not relative ones
    path = []
    for p in sys.path:
        if os.path.abspath(p) != os.path.abspath(os.getcwd()):
            path.append(p)

    import pkgutil
    names = set()
    for _, name, _ in pkgutil.iter_modules(path):
        names.add(name)
    for name in sys.builtin_module_names:
        names.add(name)

    return _cleanup_names(names)


if _PY3:
    _bytes = bytes
else:
    _bytes = str


def decode_all(obj):
    """ traverse object (recursively traversing dicts, lists, and tuples) to decode all raw strings into unicode """
    def rec(obj, seen={}):
        if isinstance(obj, _bytes):
            try:
                # first try utf-8
                return obj.decode('utf_8')
            except UnicodeDecodeError:
                pass
            # try raw_unicode_escape, which interprets the string as a Python raw unicode literal,
            # which should be quite robust
            return obj.decode('raw_unicode_escape', errors="replace")
        elif isinstance(obj, (list, dict, tuple)):
            objid = id(obj)
            if objid in seen:
                raise Exception("circular data structure")
            seen[objid] = obj

            if isinstance(obj, dict):
                out = {rec(k): rec(v) for k, v in obj.items()}
            elif isinstance(obj, list):
                out = [rec(x) for x in obj]
            elif isinstance(obj, tuple):
                out = tuple(rec(x) for x in obj)

            del seen[objid]
            return out
        return obj
    return rec(obj)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("package", help="package to explore")
    parser.add_argument("version", help="version of the package to explore")
    parser.add_argument("out", help="path to write output to")
    parser.add_argument("log", help="path to write log to")
    parser.add_argument("--toplevel", nargs="*", help="top level names to try and import for the package")
    parser.add_argument("--verbosity", nargs=1, default="INFO", help="log level")
    args = parser.parse_args()

    # Log stuff
    try:
        verbosity = int(args.verbosity)
    except:
        verbosity = getattr(logging, args.verbosity)
    logging.basicConfig(filename=args.log, level=verbosity)

    # Get dependencies before doing exploration to avoid weird edge cases
    metadata = None
    if args.package != "builtin-stdlib":
        metadata = DistMeta.from_name(args.package)
    if metadata is None:
        logger.error("unable to get metadata for {}".format(args.package))

    if args.package == "builtin-stdlib":
        # special flag for python std lib
        names = stdlib_names()
        skip = blacklisted_stdlib
    else:
        # Get top level names for the package
        names = top_level_names(metadata, from_user=args.toplevel)
        skip = blacklisted

    if len(names) == 0:
        logger.critical("unable to find any toplevel names to import\n")
        sys.exit(1)

    # Filter out blacklisted names
    filtered = sorted((name for name in names if not skip(name)), key=lambda s: s.lower())
    if len(filtered) == 0:
        logger.critical("no top level names remained after blacklist filter, started with:\n%s\n", " ".join(names))
        sys.exit(1)
    names = filtered

    # Importing arbitrary python packages tends to generate a large number of unhelpful warnings
    warnings.filterwarnings("ignore")

    # It turns out that importing certain files actually causes changes to the __builtin__ package
    # so keep a reliable reference to the stuff that we'll need
    realopen = open
    reallen = len
    realjson = json

    # Disable garbage collection since otherwise it is difficult to keep IDs consistent
    gc.disable()

    # Explore the package
    logger.info("exploring top level names: {}".format(", ".join(names)))
    shards = {}
    root_ids = {}
    for name in names:
        logger.info("walking top-level name {}".format(name))
        exploration = Explorer.explore_package(name, metadata=metadata, refmap=REFMAP, skipexplore=skip)
        if exploration is None:
            logger.error("exploration of top-level name {} failed".format(name))
            continue

        info_by_id = exploration['info_by_id']
        root_id = exploration['root_id']
        if root_id is None or root_id not in info_by_id:
            logger.error("exploration of top-level name {} failed: no root id".format(name))
            continue

        logger.info("found {} items for top-level name {}".format(reallen(exploration['info_by_id']), name))
        shards[name] = info_by_id
        root_ids[name] = root_id

    if reallen(shards) == 0:
        logger.critical("exploration failed, tried top level names: {}".format(", ".join(names)))
        sys.exit(1)

    logger.info("successfully explored top-level names: {}".format(", ".join(shards.keys())))

    # Write output
    out = {
        "pip-package": args.package,
        "pip-version": args.version,
        "shards": shards,
        "rootIDs": root_ids,
    }

    with realopen(args.out, "w+") as f:
        try:
            realjson.dump(decode_all(out), f)
        except UnicodeDecodeError as e:
            logger.critical("unable to dump output", exc_info=True)
            sys.exit(1)
    logger.info("Done exploring %s %s", args.package, args.version)


if __name__ == "__main__":
    main()
