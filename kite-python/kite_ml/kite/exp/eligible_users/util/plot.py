import matplotlib.pyplot as plt
import pandas as pd

from .helpers import EDITORS


def plot_user_hist(df: pd.DataFrame, users: pd.DataFrame, uid: str, max_editors: int = 2):
    """
    Produces a plot of one user's history

    :param df: the user histories DataFrame, as returned by data.load_user_histories()
    :param users: the users DataFrame, as returned by data.get_user_totals()
    :param uid: the user ID in question
    :param max_editors: usage history is shown for the most-used <max_editors> editors

    # TODO: this can probably be bette represented as the charts we use to show aggregate user histories
    """
    plt.figure(figsize=(16, 8))
    for_user = df[df.user_id == uid].set_index('day')

    def norm(s, offset=False):
        normed = s / for_user.total_events
        if offset:
            normed -= 0.01
        return normed

    def nwo(s):
        return norm(s, offset=True)

    ax = norm(for_user.python_events).plot(marker='o')

    legend = ['python']

    ed_pop = {}
    for ed in EDITORS:
        ed_pop[ed] = for_user[ed + '_running'].sum() / for_user.total_events.sum()
    ed_pop = pd.Series(ed_pop).sort_values(ascending=False)
    ed_pop = ed_pop[ed_pop > 0]
    print("editors:")
    print(ed_pop)

    popular_editors = list(ed_pop.index)

    for editor in popular_editors[:max_editors]:
        nwo(for_user[editor + '_plugin_installed']).plot(marker='o')
        nwo(for_user[editor + '_installed']).plot()
        nwo(for_user[editor + '_running']).plot()
        legend += [editor + ' p_installed', editor + ' installed', editor + ' running']

    # show a vertical line to mark the point at which the user started
    start_date = users.loc[uid].started
    if not pd.isnull(start_date):
        ymin, ymax = ax.get_ylim()
        ax.vlines(x=[start_date], ymin=ymin, ymax=ymax, color='r')

    plt.title("history for user " + uid)
    plt.tight_layout()
    plt.legend(legend)
    plt.show()
