#!/usr/bin/env python

# This script takes in a file that contains
# map[string][]*curation.SuggestionScore and learns a set of clusters
# for each entry in the map. For each cluster, it selects the top n
# results to represent the cluster.

import argparse
import json
import collections
from kite.topicmodeling.nmf import NMF
from kite.clustering import kmeans

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--package_list",
        help="file that contains a list of package names",
        required=True)
    parser.add_argument(
        "--raw_data",
        help="file containing the raw data, which is the output of " +
             "kite-go/curation/cmds/suggest-crawler/filter-suggestions",
        required=True)
    parser.add_argument(
        "--output",
        help="output file name (.json)",
        required=True)
    parser.add_argument(
        "--max_cluster_size",
        help="max size of a cluster", default=50)
    parser.add_argument(
        "--max_cluster_num",
        help="max number of clusters", default=30)
    parser.add_argument(
        "--top_n",
        help="number of top samples to get from each cluster (sorted by vote)",
        default=10)

    args = parser.parse_args()

    with open(args.package_list) as f:
        packages = [l.strip() for l in f]

    with open(args.raw_data) as f:
        package_documents = json.loads(f.readline())
        for p in package_documents:
            documents = [q for q in package_documents[p] if q['Source'] == 'so']
            package_documents[p] = documents

    payload = collections.defaultdict(lambda: dict({'Clusters': []}))

    for p in packages:
        if p in package_documents:
            docs = [' '.join(q['Tokens']) for q in package_documents[p]]
            w, h = NMF().decompose(docs)
            if len(w) > 0:
                clusters = kmeans.top_down_cluster(
                    w, range(len(docs)), args.max_cluster_size)
                if len(clusters) > args.max_cluster_num:
                    clusters = kmeans.cluster(w, range(len(docs)), args.max_cluster_num)
                for i, cluster in enumerate(clusters):
                    data = [package_documents[p][m] for m in cluster]
                    sorted_data = sorted(data, key=lambda k: k['ViewCount'],
                                         reverse=True)
                    payload[p]['Clusters'].append(
                        sorted_data[:min(args.top_n, len(sorted_data))])
            else:
                print('cannot factor for', p)

    payload['builtin-types'] = payload['__builtin__']
    payload['builtin-functions'] = payload['__builtin__']

    with open(args.output, 'w') as f:
        json.dump(payload, f)

if __name__ == "__main__":
    main()
