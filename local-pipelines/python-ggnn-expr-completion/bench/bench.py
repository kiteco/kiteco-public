from typing import Any, Dict, NamedTuple, Optional

import numpy as np
import pandas as pd
import subprocess


def run(cmd: str):
    return subprocess.check_output(cmd, shell=True)


class Result(NamedTuple):
    mem_median: int
    mem_max: int
    inf_time_median: float
    inf_time_max: float

    def to_dict(self) -> Dict[str, Any]:
        return {
            "mem_median": self.mem_median,
            "mem_max": self.mem_max,
            "inf_time_median": self.inf_time_median,
            "inf_time_max": self.inf_time_max,
        }


def measure(node_depth: Optional[int] = None,
            vocab_scale: Optional[float] = None,
            nodes: Optional[int] = None,
            edges: Optional[int] = None) -> Result:
    # create the model
    flags = ""
    if node_depth:
        flags += f"--node_depth={node_depth} "
    if vocab_scale:
        flags += f"--vocab_scale={vocab_scale} "
    run("rm -f benchdata/model.pb")
    run(f"python bench/gen_model.py --meta_info out/serve/metainfo.json --out_frozen_model benchdata/model.pb {flags}")

    # preprocess the feeds
    preprocess_flags = ""
    if nodes:
        preprocess_flags += f"--nodes={nodes} "
    if edges:
        preprocess_flags += f"--edges={edges} "
    run(f"./bin/preprocess-feeds --in benchdata/feeds.gob --out benchdata/feeds-preprocessed.gob {preprocess_flags}")

    # bench the model on the feeds
    res_str = run("./bin/bench-model --frozenmodel benchdata/model.pb --feedpath benchdata/feeds-preprocessed.gob")
    parts = res_str.decode("utf-8").strip().split(",")

    return Result(
        mem_median=int(parts[0]),
        mem_max=int(parts[1]),
        inf_time_median=float(parts[2]),
        inf_time_max=float(parts[3]),
    )


def depth_results():
    recs = []
    for depth in range(50, 260, 10):
        res = measure(node_depth=depth)
        print("depth: {}, res: {}".format(depth, res))
        rec = res.to_dict()
        rec["depth"] = depth
        recs.append(rec)
    df = pd.io.json.json_normalize(recs)
    df.to_csv("benchdata/results_depth.csv")


def vocab_scale_results():
    recs = []
    for scale in np.linspace(0.3, 1.5, num=20):
        res = measure(vocab_scale=scale)
        print("scale: {}, res: {}".format(scale, res))
        rec = res.to_dict()
        rec["scale"] = scale
        recs.append(rec)
    df = pd.io.json.json_normalize(recs)
    df.to_csv("benchdata/results_scale.csv")


def node_count_results():
    recs = []
    for nodes in range(400, 2000, 100):
        res = measure(nodes=nodes, edges=2100)
        print("nodes: {}, res: {}".format(nodes, res))
        rec = res.to_dict()
        rec["nodes"] = nodes
        recs.append(rec)
    df = pd.io.json.json_normalize(recs)
    df.to_csv("benchdata/results_nodes.csv")


def edge_count_results():
    recs = []
    for edges in range(400, 2000, 100):
        res = measure(nodes=400, edges=edges)
        print("edges: {}, res: {}".format(edges, res))
        rec = res.to_dict()
        rec["edges"] = edges
        recs.append(rec)
    df = pd.io.json.json_normalize(recs)
    df.to_csv("benchdata/results_edges.csv")


def main():
    depth_results()
    vocab_scale_results()
    node_count_results()
    edge_count_results()


if __name__ == "__main__":
    main()
