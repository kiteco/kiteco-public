import sys
import pkgutil
import argparse
import collections

def walk_subpackages(name, onerror=None):
    """
    Adapted from pkgutil.walk_packages source:
    https://github.com/enthought/Python-2.7.3/blob/master/Lib/pkgutil.py#L71
    """
    # Always yield the top-level package name
    yield name

    try:
        __import__(name)
    except ImportError:
        if onerror is not None:
            onerror(name)
    except Exception:
        if onerror is not None:
            onerror(name)
        else:
            raise
    else:
        path = getattr(sys.modules[name], '__path__', None) or []
        for _, name, _ in pkgutil.walk_packages(path, name+'.', onerror):
            yield name


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("package", help="top-level package to walk")
    parser.add_argument("--output", required=True, help="output for the import graph (json)")
    parser.add_argument("--failures", help="output for the list of failures")
    args = parser.parse_args()

    # It turns out that importing certain files actually causes changes to the __builtin__ package
    # so keep a reliable reference to the stuff that we'll need later when we're in sparta
    realopen = open

    failed_packages = []
    def on_pkgutil_error(pkg):
        failed_packages.append(pkg)

    with open(args.output, "w") as f:
        for name in walk_subpackages(args.package, onerror=on_pkgutil_error):
            f.write(name + "\n")

    if args.failures:
        with realopen(args.failures, "a") as f:
            for pkg in failed_packages:
                f.write(pkg + "\n")


if __name__ == "__main__":
    main()
