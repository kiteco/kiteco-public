import argparse

import pandas as pd
from sklearn.linear_model import LinearRegression


def main():
    config = parse_config("data/train.csv", "model/normalizers.json")
    data = pd.read_csv(config.input)
    data["utility"] = appraise(data, config)
    normalizers = pd.DataFrame({
        "normalizer": data.groupby("provider").apply(fit),
        "experimental": data.groupby("provider").apply(fit_experimental),
    })
    output = normalizers.T.to_json()
    with open(config.output, "w") as f:
        f.write(f"{output}\n")


def parse_config(default_input: str, default_output: str) -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=str, default=default_input)
    parser.add_argument("--output", type=str, default=default_output)
    parser.add_argument("--char_coef", type=float, default=1)
    parser.add_argument("--placeholder_coef", type=float, default=1)
    parser.add_argument("--identifier_coef", type=float, default=0)
    parser.add_argument("--keyword_coef", type=float, default=0)
    return parser.parse_args()


def fit(data: pd.DataFrame) -> float:
    return fit_col(data, "score")


def fit_experimental(data: pd.DataFrame) -> float:
    return fit_col(data, "experimental_score")


def fit_col(data: pd.DataFrame, col: str) -> float:
    if all(data["utility"] * data[col] == 0):
        return 1.0
    model = LinearRegression(fit_intercept=False)
    model.fit(data[[col]], data["utility"])
    return model.coef_[0]


def appraise(data: pd.DataFrame, config: argparse.Namespace) -> pd.Series:
    coefs = {
        "match_chars": config.char_coef,
        "match_placeholders": config.placeholder_coef,
        "match_identifiers": config.identifier_coef,
        "match_keywords": config.keyword_coef,
    }
    return sum(data[col] * coef for col, coef in coefs.items())


if __name__ == "__main__":
    main()
