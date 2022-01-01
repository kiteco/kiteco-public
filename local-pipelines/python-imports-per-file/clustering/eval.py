#!/usr/bin/env python
import json
import os
import sys
import argparse
from pprint import pprint
from typing import List, Optional, Tuple

from cluster import filter_packages, filter_imports

def eval_pkglists(filtered_imports: List[List[str]], pkglists: List[List[str]]) -> None:
    def _select_pkglist(file_imports: List[str], pkglists: List[List[str]]) -> int:
        counts = []

        # compute overlap w/ each packagelist
        for idx, pkglist in enumerate(pkglists):
            counts.append((idx, len(set(pkglist).intersection(set(file_imports)))))

        # sort and extract the index of the pkglist with highest overlap
        counts.sort(key=lambda x: x[1])
        return counts[-1][0]

    def _hit_miss(file_imports: List[str], pkglist: List[str]) -> Tuple[List[str], List[str]]:
        hits = []
        misses = []
        for pkg in file_imports:
            if pkg in pkglist:
                hits.append(pkg)
            else:
                misses.append(pkg)
        return hits, misses

    miss_map = {}
    for file_imports in filtered_imports:
        selected_pkglist_idx = _select_pkglist(file_imports, pkglists)
        hits, misses = _hit_miss(file_imports, pkglists[selected_pkglist_idx])

        if len(misses) not in miss_map:
            miss_map[len(misses)] = 0

        miss_map[len(misses)] += 1

    miss_list = [(k, v) for k, v in miss_map.items()]
    miss_list.sort(key=lambda x: x[0])

    cummulative = 0
    total = sum(count for _, count in miss_list)
    print("misses\tcount\tcummulative_pct")
    for misses, count in miss_list:
        cummulative += count
        print("%d\t%d\t%.04f" % (misses, count, float(cummulative)/float(total)))

if __name__ == "__main__":
    parser = argparse.ArgumentParser()

    parser.add_argument('--extracted_imports', type=str)
    parser.add_argument('--packagelists', type=str)

    args = parser.parse_args()

    allpkgs = []
    pkglists = []
    for packagelist in args.packagelists.split(','):
        pkglist = filter_packages(packagelist)
        pkglists.append(pkglist)
        for pkg in pkglist:
            if pkg not in allpkgs:
                allpkgs.append(pkg)

    filtered_imports = filter_imports(
        args.extracted_imports,
        allpkgs,
    )

    eval_pkglists(filtered_imports, pkglists)