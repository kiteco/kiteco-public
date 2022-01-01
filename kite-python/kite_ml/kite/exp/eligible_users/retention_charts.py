from typing import Callable

import datetime
import logging
import matplotlib.pyplot as plt
import os
import pandas as pd
import tqdm

from .util import helpers
from .util.cohort import counts_for_daily_cohort
from .retention_rate import naive_retention_fn, true_retention_fn


def monthly_retention_counts(
        histories: pd.DataFrame,
        users: pd.DataFrame,
        month_start: pd.Timestamp,
        active_days: int = 14,
        n_days: int = 135) -> pd.DataFrame:
    """
    Returns a DataFrame containing, for each day in [0, n_days), the fraction of users in the monthly cohort who are
    active/dormant/lost.
    """
    # get the retention counts for users in the monthly cohort
    start_days = pd.date_range(month_start, month_start + pd.offsets.MonthBegin(1) - datetime.timedelta(days=1))

    monthly_counts: pd.DataFrame = None  # this will be a DataFrame in the same format as counts_for_daily_cohort()

    monthly_cohort_size = 0

    for start_day in tqdm.tqdm(start_days):
        daily_counts = counts_for_daily_cohort(
            histories,
            users,
            start_day,
            active_days=active_days,
            n_days=n_days,
            cohort_column='first_py_event',  # what we care about is users who became activated on the given day
        )
        assert daily_counts.unactivated.sum() == 0, "all users in cohort should be activated"

        daily_cohort_size = daily_counts.iloc[0].sum()
        monthly_cohort_size += daily_cohort_size
        if monthly_counts is None:
            monthly_counts = (daily_counts * daily_cohort_size)
        else:
            monthly_counts += (daily_counts * daily_cohort_size)

    return monthly_counts / monthly_cohort_size


def monthly_retention_plot(filename: str,
                           histories: pd.DataFrame,
                           users: pd.DataFrame,
                           active_days: int,
                           retention_fn: Callable[[pd.DataFrame], pd.Series],
                           title: str,
                           figsize=(10, 7),
                           dpi=300):
    """
    Create a plot of retention rate (y axis) by day after first kite_status event (x axis). Multiple plots for
    different monthly cohorts are overlaid over one another.

    :param retention_fn: operates on a DataFrame containing user counts for the following columns
        - [unactivated, activate, lost, dormant] and returns a series containing just one column with the retention rate
    """
    plt.figure(figsize=figsize)

    first_day = helpers.get_first_day(histories)
    last_day = helpers.get_last_day(histories)

    # iterate over the monthly cohorts until we reach one that's close to the last day of the analysis
    month_start = first_day + pd.offsets.MonthBegin(1)
    months = []
    while month_start + pd.offsets.MonthBegin(1) + datetime.timedelta(days=15) < last_day:
        logging.debug(f"calculating retention curve for month: {month_start}")

        # calculate for how many days we should measure the retention of this cohort. The limiting factor here is that
        # there needs to be enough data for the tail end of the cohort (i.e. users that started last day of the month)
        # to be measured
        days_available = (last_day - (month_start + pd.offsets.MonthBegin(1))).days
        n_days = min(120, days_available)
        counts = monthly_retention_counts(histories, users, month_start, active_days=active_days, n_days=n_days)
        retention_fn(counts).plot()
        months.append(f"{month_start.year}-{month_start.month:02}")

        month_start += pd.offsets.MonthBegin(1)

    plt.legend(months)
    plt.xlabel("day measured")
    plt.ylabel(f"{active_days}-day retention rate")
    plt.title(title)
    plt.savefig(filename, dpi=dpi)


def retention_charts(out_dir: str, histories: pd.DataFrame, users: pd.DataFrame, win_survey: pd.DataFrame):
    true_retention_fn_14 = true_retention_fn(win_survey, 14)
    true_retention_fn_30 = true_retention_fn(win_survey, 30)
    monthly_retention_plot(
        os.path.join(out_dir, "true_14d.png"), histories, users, 14, true_retention_fn_14, "True 14-day retention")
    monthly_retention_plot(
        os.path.join(out_dir, "true_30d.png"), histories, users, 30, true_retention_fn_30, "True 30-day retention")

    monthly_retention_plot(
        os.path.join(out_dir, "naive_14d.png"), histories, users, 14, naive_retention_fn, "Naive 14-day retention")
    monthly_retention_plot(
        os.path.join(out_dir, "naive_30d.png"), histories, users, 30, naive_retention_fn, "Naive 30-day retention")






