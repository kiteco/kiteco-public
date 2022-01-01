""" Helpers for interactive traversal/inspection of the pkgexploration graph """

import json
import logging

logger = logging.getLogger(__name__)

def load_graph(path):
    with open(path) as f:
        return json.load(f)

def find_node(g, path):
    parts = path.split('.')
    index = g['shards'][parts[0]]

    cur = [x for x in index.values() if x['canonical_name'] == parts[0]]
    if len(cur) != 1:
        raise Exception('found {} choices for root node'.format(len(cur)))
    cur = cur[0]

    for part in parts[1:]:
        if part not in cur['members']:
            return None
        cur = index[str(cur['members'][part]['node_id'])]

    return cur
