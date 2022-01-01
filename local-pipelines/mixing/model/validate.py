from typing import Callable

import numpy as np
import pandas as pd

from train import parse_config, appraise


def main():
    config = parse_config("data/validate.csv", "model/validate.txt")
    data = pd.read_csv(config.input)
    data["utility"] = appraise(data, config)
    data["hit"] = (data["utility"] > 0).astype(float)
    write(
        config.output,
        mean_utility=analyze(data, np.mean, 1, 2, 5),
        total_utility=analyze(data, sum, 1, 2, 5),
        overall_rank_frequency=count(data, 5, "rank"),
        top_1_source_frequency=count(data, 1, "source"),
        top_5_source_frequency=count(data, 5, "source"),
    )


Aggregator = Callable[[pd.Series], float]


def analyze(data: pd.DataFrame, agg: Aggregator, *args: int) -> pd.DataFrame:
    hits = pd.DataFrame({f"top_{k}": top_k(data, k, "hit", agg) for k in args})
    utility = pd.DataFrame({f"top_{k}": top_k(data, k, "utility", agg) for k in args})
    metrics = pd.concat([hits, utility], axis=1)
    metrics.columns = pd.MultiIndex.from_product([
        ["hits", "utility"],
        [f"top_{k}" for k in args],
    ])
    return metrics


def top_k(data: pd.DataFrame, k: int, col: str, agg: Aggregator) -> pd.Series:
    values = (
        data[data["rank"] < k]
            .groupby(["cohort", "sample_id"])
            .max()[col]
    )
    cohorts = sorted(set(data["cohort"]))
    return pd.Series({cohort: agg(values[cohort]) for cohort in cohorts})


def count(data: pd.DataFrame, k: int, col: str) -> pd.DataFrame:
    tall = (
        data[data["rank"] < k]
            .groupby(["cohort", col])
            .count()["sample_id"]
            .reset_index()
    )
    return pd.pivot_table(tall, index="cohort", columns=col).fillna(0)["sample_id"]


def write(path: str, **kwargs: pd.DataFrame):
    labels = (label.replace("_", " ").upper() for label in kwargs.keys())
    tables = (table(data) for data in kwargs.values())
    output = "\n\n".join(f"{label}\n{table}" for label, table in zip(labels, tables))
    with open(path, "w") as f:
        f.write(f"{output}\n")


def table(data: pd.DataFrame) -> str:
    if (data.values % 1 == 0).all():
        return data.to_string(float_format="%.0f")
    return data.to_string(float_format="%.3f")


if __name__ == "__main__":
    main()
