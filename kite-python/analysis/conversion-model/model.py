import json

import matplotlib.pyplot as plt
import numpy as np
import pandas as pd
import seaborn as sns
from sklearn import linear_model, model_selection, metrics
#from sklearn import ensemble, pipeline, preprocessing


GROUPS = ["free", "trial", "pro"]
COUNTRIES = {
    "US": "USA",
    "CN": "China",
    "IN": "India",
}
DISCOUNTS = {
    "default_no_discount": 1.0,
    "25_discount": 0.75,
    "50_discount": 0.5,
}


def main():
    data, discount_data = read()

    plt.style.use("seaborn")

    roc_curves(data, out="charts/roc.png")
    pr_curves(data, out="charts/precision-recall.png")

    prob_dist(data, cumulative=False, out="charts/pdf.png")
    prob_dist(data, cumulative=True, out="charts/cdf.png")

    bars(data, col="os", out="charts/os.png")
    bars(data, col="git_found", out="charts/git.png")
    box_plot(data, col="cpu_threads", out="charts/cpu_threads.png")
    bars(data, col="geo", out="charts/geo.png")
    bars(data, col="windows_domain_membership", out="charts/wdm.png")
    bars(data, col="activation_month", out="charts/activation-month.png")
    bars(data, col="intellij_paid", out="charts/intellij_paid.png")
    editors = ("atom", "intellij", "pycharm", "sublime3", "vim", "vscode")
    for editor in editors:
        col = f"{editor}_installed"
        out = f"charts/{editor}.png"
        bars(data, col=col, out=out)

    model = train_model(data)
    write(model)

    # dynamic discount analysis:
    #roc_curves(discount_data, out="charts/discount-roc.png")
    #pr_curves(discount_data, out="charts/discount-precision-recall.png")
    #model = train_model(discount_data)
    #price(discount_data, model)
    #price(data, model)


def read():
    data = pd.read_csv("train.csv")
    data["windows"] = data["os"] == "windows"
    data["darwin"] = data["os"] == "darwin"
    data["linux"] = data["os"] == "linux"
    data["group"] = data.apply(group, axis=1)
    data["id"] = list(range(len(data)))
    data["cpu_threads"].fillna(data["cpu_threads"].mean(), inplace=True)
    for code, country in COUNTRIES.items():
        data[country] = data["country_iso_code"] == code
    data["geo"] = [
        COUNTRIES.get(c, "other or unknown")
        for c in data["country_iso_code"]
    ]
    data["discount_value"] = 0
    for discount, value in DISCOUNTS.items():
        data[discount] = data["discount"] == discount
        data["discount_value"] += data[discount] * value
    discount_data = data[data["discount"].isin(DISCOUNTS)]
    return data, discount_data


def group(row):
    if row["converted"]:
        return 2
    if row["trial_or_converted"]:
        return 1
    return 0


def box_plot(data, col, out):
    fig, ax = plt.subplots(figsize=(5, 4))
    sns.boxplot(
        x="group",
        y=col,
        data=data,
        color=sns.color_palette()[0],
        showfliers=False,
        whis=(10, 90),
        ax=ax,
    )
    ax.set_xticklabels(GROUPS)
    ax.set_xlabel("")
    ax.set_ylabel("")
    fig.tight_layout()
    fig.savefig(out)


def bars(data, col, out):

    def normalize(x):
        return x / x.sum()

    counts = data.groupby([col, "group"]).count()["id"].reset_index()
    counts["normalized"] = counts.groupby("group")["id"].transform(normalize)

    fig, ax = plt.subplots(sharex=True, figsize=(5, 4))
    sns.barplot(x="group", y="normalized", hue=col, data=counts, ax=ax)
    ax.set_xticklabels(GROUPS)
    ax.set_xlabel("")
    ax.set_ylabel("")
    ax.set_ylim(0, 1)
    fig.tight_layout()
    fig.savefig(out)


def pr_curves(data, out):
    curves(
        data,
        curve=metrics.precision_recall_curve,
        score=metrics.average_precision_score,
        flip=True,
        xlabel="recall",
        ylabel="precision",
        scorelabel="average precision score",
        out=out,
    )


def roc_curves(data, out):
    curves(
        data,
        curve=metrics.roc_curve,
        score=metrics.roc_auc_score,
        flip=False,
        xlabel="false positive rate",
        ylabel="true positive rate",
        scorelabel="area under curve",
        out=out,
    )


def curves(data, curve, score, flip, xlabel, ylabel, scorelabel, out):
    fig, ax = plt.subplots(ncols=2, figsize=(9, 4))

    feature_sets = select_features()
    scores = {}
    for name, feature_set in feature_sets.items():
        X = data[feature_set + ["discount"]].values
        y = data["converted"].values

        model = wrap_model(get_model)
        kfold = model_selection.StratifiedKFold(
            n_splits=5,
            shuffle=True,
            random_state=0,
        )

        probs = []
        labels = []
        for train, test in kfold.split(X, y):
            model.fit(X[train][:, :-1], y[train], X[train][:, -1])
            probs.extend(model.predict_proba(X[test][:, :-1], X[test][:, -1]))
            labels.extend(y[test])

        u, v, _ = curve(labels, probs)
        if flip:
            u, v = v, u
        scores[name] = score(labels, probs)
        ax[0].plot(u, v, label=name)

    ax[0].legend()
    ax[0].set_xlim(0, 1)
    ax[0].set_ylim(0, 1)
    ax[0].set_xlabel(xlabel)
    ax[0].set_ylabel(ylabel)
    ax[1].set_ylim(0, 1)

    labels = [f"{k}\n{v:.4f}" for k, v in scores.items()]
    print(labels)
    ax[1].bar(labels, scores.values(), color=sns.color_palette())
    ax[1].set_ylabel(scorelabel)
    fig.tight_layout()
    fig.savefig(out)


def prob_dist(data, cumulative, out):
    feature_sets = select_features()
    fig, axis = plt.subplots(
        ncols=len(feature_sets),
        figsize=(9, 4),
        sharex=True,
        sharey=True,
    )
    if len(feature_sets) == 1:
        axis = [axis]

    for ax, (name, feature_set) in zip(axis, feature_sets.items()):
        X = data[feature_set].values
        y = data["converted"].values

        model = wrap_model(get_model)
        kfold = model_selection.StratifiedKFold(
            n_splits=5,
            shuffle=True,
            random_state=0,
        )

        grouped_probs = {label: [] for label in (False, True)}
        for train, test in kfold.split(X, y):
            model.fit(X[train, :-1], y[train], X[train, -1])
            probs = model.predict_proba(X[test, :-1], X[test, -1])
            for prob, label in zip(probs, y[test]):
                grouped_probs[label].append(prob)

        colors = sns.color_palette()[2:]
        for color, (group, probs) in zip(colors, grouped_probs.items()):
            sns.kdeplot(
                np.log10(probs),
                label=group,
                cumulative=cumulative,
                ax=ax,
                color=color,
                legend=False,
            )
        ax.set_title(name)
        ax.set_xlabel("probability (log base 10)")

    axis[-1].legend()

    fig.tight_layout()
    fig.savefig(out)


def get_model():
    return linear_model.LogisticRegression(max_iter=1000)
    #return ensemble.GradientBoostingClassifier(n_estimators=200, max_depth=1)
    #scaler = preprocessing.StandardScaler()
    #poly = preprocessing.PolynomialFeatures(degree=2)
    #logistic = linear_model.LogisticRegression(
        #penalty="l1",
        #solver="liblinear",
        #max_iter=1000,
    #)
    #return pipeline.Pipeline([
        #("scaler", scaler),
        #("poly", poly),
        #("logistic", logistic),
    #])


def wrap_model(get_model):
    return Single(get_model)
    #return Submodels(get_model)


def select_features():
    basic = [
        "windows",
        "darwin",
        "linux",
        "git_found",
        "cpu_threads",
        "intellij_paid",
        "atom_installed",
        "intellij_installed",
        "pycharm_installed",
        "sublime3_installed",
        "vim_installed",
        "vscode_installed",
        #"selected_days",
        #"discount_value",
    ] + list(COUNTRIES.values()) #+ list(DISCOUNTS.keys())

    return {
        "basic": basic,
    }


def train_model(data):
    feature_set = select_features()["basic"]
    X = data[feature_set + ["discount"]].values
    y = data["converted"].values
    model = wrap_model(get_model)
    model.fit(X[:, :-1], y, X[:, -1])
    return model


def price(data, model):
    pr = pd.DataFrame()
    mut = data.copy()
    mut["discount_value"] = 0
    mut["discount"] = ""
    for discount in DISCOUNTS:
        mut[discount] = False
    for discount, value in DISCOUNTS.items():
        mut[discount] = True
        mut["discount_value"] = value
        mut["discount"] = discount
        feature_set = select_features()["basic"]
        X = mut[feature_set + ["discount"]].values
        pr[f"{discount}_prob"] = model.predict_proba(X[:, :-1], X[:, -1])
        pr[f"{discount}_rev"] = value * pr[f"{discount}_prob"]
        mut[discount] = False
        mut["discount_value"] = 0
        mut["discount"] = ""
    pr["max_rev"] = pr.apply(
        lambda row: max(row[f"{discount}_rev"] for discount in DISCOUNTS),
        axis=1,
    )
    pr["p0_gt_p25"] = pr["default_no_discount_prob"] > pr["25_discount_prob"]
    pr["p0_gt_p50"] = pr["default_no_discount_prob"] > pr["50_discount_prob"]
    pr["p25_gt_p50"] = pr["25_discount_prob"] > pr["50_discount_prob"]
    for discount in DISCOUNTS:
        pr[f"{discount}_is_optimal"] = pr[f"{discount}_rev"] == pr["max_rev"]
    print(pr.mean())


def write(model):
    feature_set = select_features()["basic"]
    parameters = ["intercept"] + feature_set
    values = [model.model.intercept_[0]] + list(model.model.coef_[0])
    coefficients = dict(zip(parameters, values))
    with open("params.json", "w") as fp:
        json.dump(coefficients, fp, indent=2)


class Single:
    def __init__(self, get_model):
        self.model = get_model()

    def fit(self, X, Y, groups):
        self.model.fit(X, Y)

    def predict_proba(self, X, groups):
        return self.model.predict_proba(X)[:, 1]


class Submodels:
    def __init__(self, get_model):
        self.model0 = get_model()
        self.model25 = get_model()
        self.model50 = get_model()

    def fit(self, X, Y, groups):
        X0, X25, X50 = [], [], []
        Y0, Y25, Y50 = [], [], []
        for x, y, group in zip(X, Y, groups):
            if group == "default_no_discount":
                X0.append(x)
                Y0.append(y)
                continue
            if group == "25_discount":
                X25.append(x)
                Y25.append(y)
                continue
            X50.append(x)
            Y50.append(y)
        self.model0.fit(X0, Y0)
        self.model25.fit(X25, Y25)
        self.model50.fit(X50, Y50)

    def predict_proba(self, X, groups):
        y_hat = []
        for x, group in zip(X, groups):
            if group == "default_no_discount":
                y_hat.append(self.model0.predict_proba([x])[0, 1])
                continue
            if group == "25_discount":
                y_hat.append(self.model25.predict_proba([x])[0, 1])
                continue
            y_hat.append(self.model50.predict_proba([x])[0, 1])
        return y_hat


if __name__ == "__main__":
    main()
