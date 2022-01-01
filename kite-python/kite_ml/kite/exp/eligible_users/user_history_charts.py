from typing import List, Set

import datetime
import matplotlib.pyplot as plt
from matplotlib.colors import LinearSegmentedColormap
import numpy as np
import os
import pandas as pd
import random
import seaborn as sns
import tqdm

from .util import helpers


def plot_user_history_chart(filename: str,
                            histories: pd.DataFrame,
                            user_ids: List[int],
                            start_date: pd.Timestamp,
                            n_days: int,
                            title: str,
                            figsize=(16,8),
                            dpi=300):
    # 0: unactivated (no kite_status yet)
    # 1: no kite_status
    # 2: kite_status with no python events
    # 3: kite_status with python events
    heatmap = np.zeros((len(user_ids), n_days))

    for u, uid in enumerate(tqdm.tqdm(user_ids)):
        uh = histories[histories.user_id == uid]
        first_day = uh.day.min()

        for d in range(n_days):
            day = pd.Timestamp(start_date) + datetime.timedelta(days=d)
            if day < first_day:
                # user hasn't had a kite_status event yet
                heatmap[u, d] = 0
                continue
            on_day = uh[uh.day == day]
            if len(on_day) == 0:
                # no kite_status for that user/day
                heatmap[u, d] = 1
                continue
            on_day = on_day.iloc[0]
            if on_day.python_events > 0:
                heatmap[u, d] = 3
            else:
                heatmap[u, d] = 2

    plt.figure(figsize=figsize)

    colors = ((255, 255, 255),  # unactivated
              (181, 110, 110),  # no kite_status
              (166, 206, 227),  # kite_status, no python
              (124, 181, 100),  # python
              )
    colors = [(c[0]/256, c[1]/256, c[2]/256) for c in colors]

    cmap = LinearSegmentedColormap.from_list('Custom', colors, len(colors))

    sns.heatmap(heatmap[:76, :80], linewidth=1, cmap=cmap, cbar=False)
    plt.title(title)
    plt.xlabel("day, starting at {}".format(start_date.date()))
    plt.ylabel("user")
    plt.savefig(filename, dpi=dpi)


def user_history_charts(out_dir: str,
                        histories: pd.DataFrame,
                        users: pd.DataFrame,
                        active_days: int = 14,
                        n_days: int = 70,
                        users_to_show: int = 30):
    """
    Produces charts showing history for random samplings of users.
    :param out_dir: directory in which to write the plots
    :param histories: DataFrame as returned by load_user_histories()
    :param users: DataFrame as returned by get_user_totals()
    :param active_days: the "n" in "n-day active"
    :param n_days: the number of days for which to plot the history
    :param users_to_show: the number of users to sample for each plot
    """
    last_day = helpers.get_last_day(histories)
    first_day = last_day - datetime.timedelta(days=n_days)

    cutoff = last_day - datetime.timedelta(days=active_days)

    # limit plot to users who had at least one python event in the plot window
    valid_user_ids = set(histories[
        (histories.day > first_day) & (histories.day < last_day) & (histories.python_events > 0)].user_id)
    valid_users = users[users.index.isin(valid_user_ids)]

    kite_status_users = set(valid_users[valid_users.last_day >= cutoff].index)
    active_users = set(valid_users[valid_users.last_py_event >= cutoff].index)
    dormant_users = kite_status_users.difference(active_users)

    # to find recently lost users, we find users who became lost within 10 days of the cutoff
    lost_users = set(valid_users[(valid_users.last_day < cutoff) &
                                 (valid_users.last_day >= cutoff - datetime.timedelta(days=10))].index)

    md_users = set(valid_users[(valid_users.metrics_disabled == True) & (valid_users.last_day < last_day)].index)

    random.seed(100)

    def plot(filename: str, title: str, users: Set[int]):
        plot_users = sorted(random.sample(users, users_to_show))
        filename = os.path.join(out_dir, filename)
        plot_user_history_chart(filename, histories, plot_users, first_day, n_days, title)

    plot("active.png", "active users", active_users)
    plot("dormant.png", "dormant users", dormant_users)
    plot("lost.png", "recently lost users", lost_users)
    plot("metrics_disabled.png", "users who disabled metrics", md_users)
    plot("all_users.png", "random sampling of all users", valid_user_ids)
