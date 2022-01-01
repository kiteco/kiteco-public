from typing import Dict, List, Tuple, Optional, NamedTuple

import argparse
import matplotlib
import math
matplotlib.use('Agg')

import matplotlib.pyplot as plt
import os
import pandas as pd

from analysis.data import read_logs, daily_user_agg, AggType


def plot_series_to_file(filename: str,
                        series: Dict[str, pd.Series],
                        title: str = '',
                        ylabel: str = '',
                        show_y_axis: bool = True,
                        figsize: Tuple[int, int] = (12, 6),
                        dpi: int = 300):
    fig = plt.figure(figsize=figsize)
    ax = fig.gca()
    plot_series(ax, series, title, ylabel, show_y_axis=show_y_axis)
    fig.savefig(filename, dpi=dpi)


def plot_series(ax: plt.Axes,
                series: Dict[str, pd.Series],
                title: str = '',
                ylabel: str = '',
                show_y_axis: bool = True):
    keys = sorted(series.keys())

    plotted = []

    for k in keys:
        if len(series[k]) > 0:
            series[k].plot(ax=ax, marker='o')
            plotted.append(k)
    if len(plotted) > 0:
        ax.legend(plotted)

    ax.set_title(title)
    ax.set_ylabel(ylabel)
    ax.get_yaxis().set_visible(show_y_axis)


def weekly_mean(s: pd.Series) -> pd.Series:
    return s.resample('W').mean()


def plot_daily_engaged_users(
        ax: plt.Axes,
        df: pd.DataFrame,
        metric: str,
        denominator: Optional[str],
        agg_type: AggType):
    """
    Produces a plot showing daily counts of a particular metric aggregated over all users and over all
    engaged (>=10 completion events) users

    :param df: as returned by read_logs()
    :param denominator: if present, the metric is divided by this metric for each user/day
    """
    title = f"{agg_type.value} {metric}"
    metric_fn = lambda df: df[metric]
    if denominator:
        title += f" / {denominator}"
        metric_fn = lambda df: df[metric] / df[denominator]

    series = {
        'all': daily_user_agg(df, metric_fn, agg_type, min_user_events=0),
        'engaged': daily_user_agg(df, metric_fn, agg_type, min_user_events=10),
    }
    plot_series(ax, series, title=title)


def plot_multiple_user_aggregations_to_file(
        filename: str,
        df: pd.DataFrame,
        metrics: List[List[str]],
        agg_type: AggType,
        dpi: int = 300):
    """
    Produces plots for a bunch of metrics side by side, aggregated over all users.

    :param df: as returned by read_logs()
    :param metrics: A list of lists describing the metrics to plot. Each list within can either be of size 1
        or of size 2; in the latter case, the second metric is interpreted as a denominator.
    """
    n_rows = int(math.ceil(len(metrics) / 2))
    fig = plt.figure(figsize=(16, n_rows*4))
    axs = fig.subplots(nrows=n_rows, ncols=2)

    for i, metric in enumerate(metrics):
        row = i // 2
        col = i % 2
        ax = axs[row, col]
        plot_daily_engaged_users(ax, df,
                                 metric=metric[0],
                                 denominator=metric[1] if len(metric) > 1 else None,
                                 agg_type=agg_type)
    fig.tight_layout()
    fig.savefig(filename, dpi=dpi)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--logs', type=str, required=True)
    parser.add_argument('--plots_dir', type=str, default='./plots')

    args = parser.parse_args()

    df = read_logs(args.logs)

    df.info()

    def plotpath(filename: str) -> str:
        return os.path.join(args.plots_dir, filename)

    all_metrics = [
        ['total_events'],
        ['requested_expected'],
        ['shown'],
        ['shown', 'requested_expected'],
        ['at_least_one_shown'],
        ['at_least_one_shown', 'requested_expected'],
        ['shown', 'total_events'],
        ['requested_expected', 'total_events'],
        ['selected_num'],
        ['selected_num', 'requested_expected'],
    ]

    plot_multiple_user_aggregations_to_file(
        plotpath('completions_median.png'),
        df,
        all_metrics,
        AggType.MEDIAN)

    plot_multiple_user_aggregations_to_file(
        plotpath('completions_mean.png'),
        df,
        all_metrics,
        AggType.MEAN)

    mtac_metrics = [
        ['at_least_one_shown_call_model', 'at_least_one_shown'],
        ['selected_2_call_model', 'at_least_one_shown_call_model'],
        ['selected_2_mtac', 'at_least_one_shown_mtac'],
        ['selected_2_call_model', 'at_least_one_shown'],
        ['selected_2_mtac', 'at_least_one_shown'],
        ['selected_2_call_model', 'requested_expected'],
        ['selected_2_mtac', 'requested_expected'],
    ]

    plot_multiple_user_aggregations_to_file(
        plotpath('mtac_mean.png'),
        df[df.index > '2019-05-01'],
        mtac_metrics,
        AggType.MEAN,
    )

    plot_series_to_file(
        plotpath('attr.png'),
        {'attr': daily_user_agg(df, lambda df: df.selected_2_attribute_model, AggType.MEAN)},
        title="Mean attribute model completions selected per day")

    plot_series_to_file(
        plotpath("call.png"),
        {'call': daily_user_agg(df, lambda df: df.selected_2_call_model, AggType.MEAN)},
        title="Mean call completions selected per day")

    plot_series_to_file(
        plotpath("shown.png"),
        {'shown': daily_user_agg(df, lambda df: df.shown, AggType.MEDIAN)},
        title="Median completions shown per day")


if __name__ == "__main__":
    main()
