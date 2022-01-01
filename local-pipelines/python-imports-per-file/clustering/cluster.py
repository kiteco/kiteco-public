#!/usr/bin/env python
import json
import os
import sys
import argparse
from pprint import pprint
from typing import List, Optional

import numpy as np
import matplotlib.pyplot as plt
from numpy.linalg import svd
from stdlib_list import stdlib_list

def pca(imports: List[str], pkglist: List[str]) -> None:
    pkglist.sort()

    idx_to_pkg = {k: v for k, v in enumerate(pkglist)}
    pkg_to_idx = {v: k for k, v in idx_to_pkg.items()}

    A = np.zeros((len(pkglist), len(pkglist)))

    for file_imports in imports:
        for pkg1 in file_imports:
            pkg1_idx = pkg_to_idx[pkg1]
            for pkg2 in file_imports:
                pkg2_idx = pkg_to_idx[pkg2]
                A[pkg1_idx, pkg2_idx] += 1

    A = (A - np.mean(A, axis=0)) / np.std(A, axis=0)

    U, S, Vt = svd(A)
    k = 2
    Z = np.diag(S[:k])

    fig, ax = plt.subplots()

    ax.plot(Z[:,0], Z[:,1], 'ro')
    for idx, xy in enumerate(zip(Z[:,0], Z[:,1])):
        ax.annotate(idx_to_pkg[idx], xy)

    plt.show()


def filter_packages(pkglist_file: str, exclude_stdlib: Optional[bool] = False, only_stdlib: Optional[bool] = False) -> List[str]:
    """
    Takes in a path to the package list and generates the set of packages to use for the
    clustering analysis.
    """

    if exclude_stdlib and only_stdlib:
        raise ValueError("cannot set both --exclude_stdlib and --only_stdlib")

    pkgs = []
    with open(pkglist_file) as lines:
        for line in lines:
            pkgs.append(line.strip())

    print("packagelist contains %d packages" % len(pkgs))

    if exclude_stdlib:
        pkgs = [pkg for pkg in pkgs if pkg not in stdlib_list("3.5")]
        print("after removing stdlib, packagelist contains %d packages" % len(pkgs))

    if only_stdlib:
        pkgs = [pkg for pkg in pkgs if pkg in stdlib_list("3.5")]
        print("after including only stdlib, packagelist contains %d packages" % len(pkgs))

    pkgs.sort()

    return pkgs


def filter_imports(imports_file: str, pkglist: List[str]) -> List[List[str]]:
    """
    Reads in a file with a json list object per line (list of imports per file),
    and applies pkglist as a filter to subselect the packages we are interestd in for
    clustering analysis.
    """

    imports = []
    with open(imports_file) as lines:
        for line in lines:
            file_imports = json.loads(line)
            file_imports = [i for i in file_imports if i in pkglist]
            if len(file_imports) > 0:
                imports.append(file_imports)

    return imports

if __name__ == "__main__":
    parser = argparse.ArgumentParser()

    parser.add_argument('--packagelist', type=str)
    parser.add_argument('--extracted_imports', type=str)
    parser.add_argument('--exclude_stdlib', action='store_true')
    parser.add_argument('--only_stdlib', action='store_true')

    args = parser.parse_args()

    pkglist = filter_packages(
        args.packagelist,
        exclude_stdlib = args.exclude_stdlib,
        only_stdlib = args.only_stdlib,
    )

    filtered_imports = filter_imports(
        args.extracted_imports,
        pkglist,
    )

    pca(filtered_imports, pkglist)
