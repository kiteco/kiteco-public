from typing import Set

import datetime
import pandas as pd
import tqdm


def _get_cohort(users: pd.DataFrame, start_date: pd.Timestamp, col_to_use: str) -> Set[str]:
    return set(users[(users[col_to_use] >= start_date) &
                     (users[col_to_use] < (start_date + datetime.timedelta(days=1)))].index)


def counts_for_daily_cohort(histories: pd.DataFrame,
                            users: pd.DataFrame,
                            start_date: pd.Timestamp,
                            active_days: int = 7,
                            n_days: int = 30,
                            cohort_column: str = 'first_day',
                            ) -> pd.DataFrame:
    """
    Produces daily counts of users by category in a daily cohort that started on <start_date>. The categories are:

    unactivated: users who have never had a Python event up to the given day.
    active: users who, on a given day, had a Python event within the past <active_days>.
    dormant: users who have had a kite_status even with the past <active_days>, but have not had a Python event within
        the past <active_days>.
    lost: users who have not had any kite_status events within the past <active_days>.

    :param histories: user histories DataFrame, as returned by load_user_histories()
    :param users: users DataFrame, as returned by get_user_totals()
    :param start_date: the start date of the daily cohort
    :param active_days: the "n" in "n-d active"
    :param n_days: the number of days after the cohort start date for which to measure the counts
    :param cohort_column: the column in the users DataFrame to use when deciding how to pick out a cohort.
        Reasonable options are:
            first_day: the first kite_status event seen for the user
            first_py_event: the first Python event seen for the user (i.e. if this is used, the cohort is defined
                as the users who had their first Python event on <start_date>)
    """
    cohort = _get_cohort(users, start_date, cohort_column)

    cohort_df = histories[histories.user_id.isin(cohort)]

    activated_users = set()

    d = {}

    for i, day in enumerate(range(n_days)):
        day = start_date + datetime.timedelta(days=i)

        # contains users who have had kite_status events within active_days
        ks_within_ad = cohort_df[
            (cohort_df.day >= day - datetime.timedelta(days=active_days-1)) &
            (cohort_df.day < day + datetime.timedelta(days=1))
        ]
        ks_within_ad_users = set(ks_within_ad.user_id)

        # contains users who have had python events within active_days
        py_within_ad = ks_within_ad[ks_within_ad.python_events > 0]
        py_within_ad_users = set(py_within_ad.user_id)

        lost_users = cohort.difference(ks_within_ad_users)

        # python events on the current day
        py_on_day_users = set(py_within_ad[py_within_ad.day >= day].user_id)

        activated_users.update(py_on_day_users)

        dormant_or_unactivated = ks_within_ad_users.difference(py_within_ad_users)
        dormant_users = dormant_or_unactivated.intersection(activated_users)
        unactivated_users = dormant_or_unactivated.difference(activated_users)

        tou = {
            'unactivated': len(unactivated_users),
            'active': len(py_within_ad_users),
            'dormant': len(dormant_users),
            'lost': len(lost_users),
        }

        for k, v in tou.items():
            if k not in d:
                d[k] = {}
            d[k][i] = v

    return pd.DataFrame(d)


def weekly_retention_counts(histories: pd.DataFrame, users: pd.DataFrame,
                            start_date: pd.Timestamp, end_date: pd.Timestamp, on_day: int = 14,
                            active_days: int = 7) -> pd.DataFrame:
    """
    Returns categorized retention counts (measured on <on_day>) for weekly cohorts
    starting at <start_date> and ending at <end_date>. See counts_for_daily_cohort() for the definitions of the
    retention categories.
    """
    if active_days >= on_day:
        raise ValueError("retention_day should be higher than active_days")

    d = {}

    for week_start in tqdm.tqdm(pd.date_range(start_date, end_date, freq='W')):
        wc = None
        for i in range(7):
            day = week_start + datetime.timedelta(days=i)
            c = counts_for_daily_cohort(histories, users, day, active_days=active_days, n_days=on_day + 1)
            if wc is None:
                wc = c
            else:
                wc += c

        rec = wc.loc[on_day]

        tou = {
            'total': rec.unactivated + rec.active + rec.dormant + rec.lost,
            'unactivated': rec.unactivated,
            'active': rec.active,
            'dormant': rec.dormant,
            'lost': rec.lost,
        }
        for k, v in tou.items():
            if k not in d:
                d[k] = {}
            d[k][week_start] = v

    return pd.DataFrame(d)
