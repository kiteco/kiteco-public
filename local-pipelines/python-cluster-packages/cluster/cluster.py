from typing import Any, NamedTuple, List, Dict, Tuple

import argparse
import numpy as np
import json
import logging
import sklearn.cluster

from kite.asserts.asserts import FieldValidator

logging.getLogger().setLevel(logging.INFO)


class ScoredCoOccurrence(NamedTuple):
    pkg1: str
    pkg2: str
    score: int

    @classmethod
    def from_json(cls, d: dict) -> 'ScoredCoOccurrence':
        v = FieldValidator(cls, d)
        return ScoredCoOccurrence(
            pkg1=v.get('pkg1', str),
            pkg2=v.get('pkg2', str),
            score=v.get('score', int),
        )


def load_scored_co_occurrences(path: str) -> List[ScoredCoOccurrence]:
    with open(path, 'r') as f:
        scored: List[ScoredCoOccurrence] = []
        for s in json.load(f):
            scored.append(ScoredCoOccurrence.from_json(s))
        return scored


def build_co_occurrence_matrix(scored: List[ScoredCoOccurrence]) -> Tuple[np.ndarray, Dict[int, str], Dict[str, int]]:
    pkg_to_idx: Dict[str, int] = dict()
    idx_to_pkg: Dict[int, str] = dict()
    for s in scored:
        for pkg in [s.pkg1, s.pkg2]:
            if pkg in pkg_to_idx:
                continue
            idx = len(pkg_to_idx)
            pkg_to_idx[pkg] = idx
            idx_to_pkg[idx] = pkg

    m: np.ndarray = np.zeros((len(pkg_to_idx), len(pkg_to_idx)), np.float)
    for s in scored:
        i = pkg_to_idx[s.pkg1]
        j = pkg_to_idx[s.pkg2]
        m[i, j] = float(s.score)

    assert np.all(np.isclose(m - m.transpose(), np.zeros_like(m, np.float)))
    return m, idx_to_pkg, pkg_to_idx


def manual_clustering(centers: List[int], co_occurs: np.ndarray, idx_to_pkg: Dict[int, str]) -> List[List[str]]:
    # just do nearest neighbor clustering to centers based on cosine similarity

    # normalize rows to l2 unit length
    co_occurs = co_occurs / np.sqrt(np.sum(co_occurs**2, axis=1, keepdims=True))

    clustered: List[List[str]] = [[] for _ in range(len(centers))]
    for p in range(len(idx_to_pkg)):
        vi = co_occurs[p, :]
        max_cos = 0
        max_center = -1
        for i, ci in enumerate(centers):
            vc = co_occurs[ci, :]
            cos = np.sum(vi * vc)
            if cos > max_cos:
                max_cos = cos
                max_center = i
        clustered[max_center].append(idx_to_pkg[p])

    # do some manual cleanup
    def move(original: int, new: int, pkg: str):
        clustered[original].remove(pkg)
        clustered[new].append(pkg)

    # tensorflow, seaborn, cv2 not in the top 100 packages so just toss them into the numpy cluster
    clustered[2].append('tensorflow')
    clustered[2].append('seaborn')
    clustered[2].append('cv2')

    # move sys, shutil, dateutil into the os cluster
    move(1, 0, 'sys')
    move(1, 0, 'shutil')
    move(1, 0, 'dateutil')

    # put __builtin__ into the os cluster
    clustered[0].append('__builtin__')

    # move nose, pickle, cPickle into the os cluster
    move(2, 0, 'nose')
    move(2, 0, 'pickle')
    move(2, 0, 'cPickle')

    return clustered


def automatic_clustering(co_occurs: np.ndarray, idx_to_pkg: Dict[int, str], clustering: Any) -> List[List[str]]:
    # scale cols by IDF
    n = float(len(idx_to_pkg))
    idfs = (co_occurs > 0).astype(np.float).sum(axis=0)
    idfs = np.log(n / idfs + 1e-6)
    idfs = co_occurs / idfs

    # normalize rows to l2 unit length
    # co_occurs = tfs * idfs
    co_occurs = co_occurs * idfs
    co_occurs = co_occurs / np.sqrt(np.sum(co_occurs**2, axis=1, keepdims=True))

    logging.info('clustering...')

    cluster = clustering.fit_predict(co_occurs)
    clustered: List[List[str]] = []
    for i in list(set(cluster.labels_)):
        members = [idx_to_pkg[j] for j in np.where(cluster == i)[0]]
        clustered.append(members)
    return clustered


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--endpoint', type=str, default='http://ml-training-0.kite.com:3039')
    parser.add_argument('--cooccurs', type=str, default='cooccurs.json')
    parser.add_argument('--num_clusters', type=int, default=5)
    parser.add_argument('--out', type=str, default='clusters.json')
    args = parser.parse_args()

    scores = load_scored_co_occurrences(args.cooccurs)
    co_occurs, idx_to_pkg, pkg_to_idx = build_co_occurrence_matrix(scores)

    centers = ['os', 'django', 'numpy']
    center_idxs = [pkg_to_idx[c] for c in centers]

    clustered_manual = manual_clustering(center_idxs, co_occurs, idx_to_pkg)
    # Or use auto clustering
    # clustering = sklearn.cluster.Birch(n_clusters=args.num_cluster)
    # clustered_auto = automatic_clustering(co_occurs, idx_to_pkg, clustering)

    with open(args.out, 'w') as f:
        json.dump(clustered_manual, f)


if __name__ == '__main__':
    main()
