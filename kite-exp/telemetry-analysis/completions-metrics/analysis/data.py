from typing import Callable, Dict

import datetime
from enum import Enum
import json
import numpy as np
import pandas as pd
import tqdm


def read_logs(filename: str) -> pd.DataFrame:
    """
    Read completions metrics from a JSON file (as produced by digest-comp-logs)

    :return: A DataFrame where each row represents the completion metrics for one kite_status event for a given user.
    The index is the timestamp of the event.
    """
    with open(filename, 'r') as f:
        lines = f.readlines()

    records = []
    for line in tqdm.tqdm(lines):
        records.append(_process_record(json.loads(line)))
    df = pd.io.json.json_normalize(records)

    df['timestamp'] = pd.to_datetime(df['timestamp'])
    df = df.set_index('timestamp')

    return df


def _process_record(rec: Dict) -> Dict:
    out = {
        'total_events': 1,

        'user_id': rec['user_id'],
        'timestamp': rec['timestamp'],
        'client_version': rec['client_version'],

        'shown': rec['completions_shown'],
        'at_least_one_shown': rec['completions_at_least_one_shown'],
        'at_least_one_new_shown': rec['completions_at_least_one_new_shown'],
        'triggered': rec['completions_triggered'],
        'requested': rec['completions_requested'],
        'requested_expected': rec['completions_requested_expected'],
        'requested_unexpected': rec['completions_requested_unexpected'],
        'requested_error': rec['completions_requested_error'],
    }

    for breakdown in ["selected", "selected_2", "completed", "at_least_one_shown"]:
        mtac_count = 0
        total_count = 0

        for typ in ["attribute_model", "traditional", "call_model", "keyword_model", "expr_model"]:
            bd = rec[f"completions_{breakdown}_by_source"]
            if not bd:
                bd = {}
            count = bd.get(typ, 0)
            total_count += count
            if typ in ("call_model", "expr_model"):
                mtac_count += count
            out[f"{breakdown}_{typ}"] = count

        out[f"{breakdown}_mtac"] = mtac_count
        out[f"{breakdown}_num"] = total_count

    return out


class AggType(Enum):
    MEAN = 'mean'
    MEDIAN = 'median'


def limit_to_weekdays(df: pd.DataFrame) -> pd.DataFrame:
    return df[~df.index.weekday.isin((5, 6))]


def daily_user_agg(
        df: pd.DataFrame,
        metric_fn: Callable[[pd.DataFrame], pd.Series],
        agg_type: AggType,
        min_user_events: int = 10) -> pd.Series:
    """
    Produces a daily timeseries of the given metric, aggregated for each day by user. Only weekdays are included.

    :param df: the completions metrics DataFrame, as returned by read_logs()
    :param metric_fn: a function mapping the completions dataframe to a series representing the desired metric
        e.g. lambda df: df.at_least_one_shown
    :param agg_type: how to aggregate the users
    :param min_user_events: for each day, limit the aggregation to those users that had at least this many events
        in that day
    """
    daily = limit_to_weekdays(df.resample('D').sum())
    days = list(daily.index)

    r: Dict[pd.Timestamp, float] = {}

    for day in days:
        day_end = day + datetime.timedelta(days=1)

        # get all events for the given day
        for_day = df[(df.index >= day) & (df.index < day_end)]

        # limit the events to those coming from users that had at least user_n_events in that day
        counts_by_user = for_day.groupby(['user_id']).size()
        engaged_users = set(counts_by_user[counts_by_user >= min_user_events].index)
        for_day = for_day[for_day.user_id.isin(engaged_users)]

        # add up stats by day by user
        by_date_user = for_day.groupby([pd.Grouper(freq='D'), 'user_id']).sum()

        # select the series representing the metric we're interested in
        metric_series = metric_fn(by_date_user)

        # remove inf values (which may have resulted from division by zero)
        metric_series = metric_series.replace([np.inf, -np.inf], np.nan).dropna()

        # for each day, group the metric for all users by the desired aggregation function
        grouped = metric_series.groupby(level=['timestamp'])
        if agg_type == AggType.MEAN:
            user_agg = grouped.mean()
        elif agg_type == AggType.MEDIAN:
            user_agg = grouped.median()
        else:
            raise ValueError("unknown agg type: {}".format(agg_type))

        # at this point we should have a series that has either one value for the given day (or zero if there are
        # no users)
        assert len(user_agg) in {0, 1}
        r[day] = user_agg.sum()

    # throw away the last day since it may be incomplete
    return pd.Series(r)[:-1]
