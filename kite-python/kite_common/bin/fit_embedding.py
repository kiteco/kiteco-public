import math
import json
import argparse
import collections

import numpy as np
import sklearn.manifold

import kite.canonicalization.utils as utils
import kite.canonicalization.discovery as discovery
import kite.canonicalization.ontology as ontology

def point_to_dict(p):
    return {'x': p[0], 'y': p[1]}


def normalize(x):
    mins = np.min(x, axis=0)
    maxs = np.max(x, axis=0)
    denom = np.maximum(maxs - mins, 1.)
    return 2. * (x - mins) / denom - 1.


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--gamma', type=float, default=-1.)
    parser.add_argument('--limit', type=int, required=False)
    parser.add_argument('errors')
    parser.add_argument('output')
    args = parser.parse_args()

    if args.gamma >= 0:
        print('Gamma must be < 0')

    # Load error messages
    strings = list(map(str.strip, open(args.errors)))

    # Categorize by class
    strings_by_class = collections.defaultdict(list)
    for s in strings:
        pos = s.find(':')
        if pos != -1:
            strings_by_class[s[:pos]].append(s)

    anchors = []
    global_nodes = []
    for i, (errorclass, strings) in enumerate(strings_by_class.items()):
        print('Processing %s (%d of %d)' % (errorclass, i, len(strings_by_class)))

        if len(strings) < 5:
            positions = np.random.randn(len(strings), 2) * .5
        else:
            # Limit number of strings
            if args.limit and len(strings) > args.limit:
                strings = [strings[i] for i in np.linspace(0, len(strings)-1, args.limit).round().astype(int)]

            # Tokenize
            tokenvecs = list(map(discovery.tokenize, strings))

            for tokenvec, s in zip(tokenvecs, strings):
                if len(tokenvec) > 20:
                    tokenvec[:] = tokenvec[:20]

            # Construct an affinity matrix
            affinity = np.eye(len(tokenvecs))
            for i, tokens in enumerate(tokenvecs):
                if (i+1) % 100 == 0:
                    print('Generating row %d of %d' % ((i+1), len(tokenvecs)))

                for j in range(i):
                    dist = discovery.compute_edit_distance(tokenvecs[i], tokenvecs[j])
                    af = math.exp(args.gamma*dist*dist)
                    affinity[i, j] = af
                    affinity[j, i] = af

            # Fit embedding
            embedding = sklearn.manifold.SpectralEmbedding(affinity='precomputed')
            positions = embedding.fit_transform(affinity)
            positions = normalize(positions)

        num_samples = 0
        anchor = np.random.randn(2) * 40.
        if len(anchors) > 0:
            while np.min(np.linalg.norm(anchor - anchors, axis=1)) < 5.:
                anchor = np.random.randn(2) * 40.
                num_samples += 1
                if num_samples > 50:
                    print('Failed to find anchor after 50 samples')
                    break
            print('Found solution after %d samples' % num_samples)
            anchors = np.vstack((anchors, anchor))
        else:
            anchors = np.array([anchor])

        positions += anchor

        # Construct nodes
        nodes = [{'label': label, 'position': point_to_dict(pt), 'pattern_id': i}
            for label, pt in zip(strings, positions)]

        global_nodes.extend(nodes)

    data = {'nodes': global_nodes}
    json.dump(data, open(args.output, 'w'), indent=2)

if __name__ == '__main__':
    import cProfile, pstats
    pr = cProfile.Profile()
    pr.enable()
    try:
        main()
    finally:
        pr.disable()
        pstats.Stats(pr).sort_stats('tottime').print_stats(12)
