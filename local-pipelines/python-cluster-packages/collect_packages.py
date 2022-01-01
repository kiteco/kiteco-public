from typing import Dict, List
import argparse
import networkx as nx
import json
import copy

from cluster.cluster import ScoredCoOccurrence


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--deps_path', type=str)
    parser.add_argument('--dist_path', type=str)
    parser.add_argument('--freq_path', type=str)
    parser.add_argument('--cooc_path', type=str)
    parser.add_argument('--ggnn_path', type=str)
    parser.add_argument('--stan_path', type=str)
    parser.add_argument('--ordered_path', type=str)
    parser.add_argument('--all_path', type=str)

    args = parser.parse_args()

    freq = set([l.strip() for l in open(args.freq_path).readlines()])
    ggnn = set([l.strip() for l in open(args.ggnn_path).readlines()])
    stan = [l.strip() for l in open(args.stan_path).readlines()]

    # deps.json is a map from a packages to a list of packages it depends on
    with open(args.deps_path) as f:
        deps: Dict[str, List[str]] = json.load(f)

    # dist.json is a map from a dist name to a list of module names
    with open(args.dist_path) as f:
        alias: Dict[str, List[str]] = json.load(f)

    cooc = set()
    with open(args.cooc_path) as f:
        data = json.load(f)
        for item in data:
            c = ScoredCoOccurrence.from_json(item)
            cooc.add(c.pkg1)
            cooc.add(c.pkg2)

    # get the package set we might care about
    packages = cooc.intersection(freq.union(ggnn))

    # Build a graph with the packages and all their dependencies
    g = nx.DiGraph()
    size = 0
    for p in packages:
        g.add_node(p)
    new_size = len(g.nodes())
    while size != new_size:
        original = copy.deepcopy(g.nodes())
        for n in original:
            for d in deps[n]:
                if d in alias:
                    ds = alias[d]
                else:
                    ds = [d]
                for dd in ds:
                    g.add_node(dd)
                    g.add_edge(n, dd)
        new_size = len(g.nodes())
        size = new_size

    # Retrieve all the packages in order, starting from the standard list
    # Remove the nodes sequentially from g until we get a full list
    final = ['__builtin__']
    final += stan
    g_trim = copy.deepcopy(g)
    while len(g_trim.nodes()) > 0:
        g_new = copy.deepcopy(g_trim)
        for n in g_trim.nodes():
            if g_trim.out_degree(n) == 0:
                g_new.remove_node(n)
                if n not in final:
                    final.append(n)
        if len(g_new.nodes()) == len(g_trim.nodes()):
            break
        g_trim = copy.deepcopy(g_new)

    # Write the ordered package list to file
    o = open(args.ordered_path, 'w+')
    for item in final:
        o.write(item + '\n')
    o.close()

    # Collect a list of all possible packages
    all_packages = cooc.union(freq).union(ggnn).union(set(stan))
    for p in deps:
        all_packages.add(p)
        for d in deps[p]:
            all_packages.add(d)

    all_o = open(args.all_path, 'w+')
    for item in all_packages:
        all_o.write(item + '\n')
    all_o.close()


if __name__ == '__main__':
    main()
