from typing import Any, Dict, List, Optional

import datetime
import json
import logging
import os
import pandas as pd
import tqdm

from ..util.helpers import EDITORS

BLANK_TIME = "0001-01-01T00:00:00Z"


def get_user_totals(histories: pd.DataFrame) -> pd.DataFrame:
    """Get dataframe of totals for each user
    :param histories: as returned by load_user_histories()
    """

    agg = dict.fromkeys(histories, 'sum')

    del agg['user_id']
    # we want to get both the first and last day in which the user was seen
    agg['day'] = ['min', 'max']
    # the user's last seen timezone is probably the most relevant one
    agg['time_zone_offset'] = 'last'
    # we don't expect these columns to change by day since these were stored on the user level
    # in the user histories
    for col in ['install_id',
                'started',
                'channel',
                'email',
                'metrics_disabled',
                'os',
                'memory_total',
                'cpu_mhz',
                'last_surveyed_time',
                'last_surveyed_profession',
                'last_surveyed_experience',
                'last_surveyed_project_types',
                'last_surveyed_company_size',
                'last_login',
                'last_seen']:
        agg[col] = 'first'

    # we group by all the columns that are the same for each user since we don't want to aggregate these
    users = histories.groupby(['user_id']).agg(agg)

    cols = []
    for col in users.columns:
        if col == ('day', 'min'):
            c = 'first_day'
        elif col == ('day', 'max'):
            c = 'last_day'
        else:
            c = col[0]
        cols.append(c)
    users.columns = cols

    py_events = histories[histories.python_events > 0].groupby(['user_id']).agg({'day': ['min', 'max']})
    users['first_py_event'] = py_events['day', 'min']
    users['last_py_event'] = py_events['day', 'max']

    return users


def load_user_histories(filename: str, pickled: Optional[str]) -> pd.DataFrame:
    """
    :param filename: the daily user histories in JSON format, as returned by the join-mixpanel command.
    :param pickled: if path is present, and the file doesn't exist, user histories will be pickled to this file.
       If the file does exist, this histories will be loaded from this file instead of from the original JSON file.
    :return: Pandas DataFrame of daily user histories
    """
    if pickled:
        if os.path.exists(pickled):
            logging.info(f"loading pickled user histories from {pickled}")
            return pd.read_pickle(pickled)
        logging.info(f"pickled histories not found in {pickled}")

    logging.info(f"loading user histories from {filename}")
    fp = open(filename, 'r')
    user_recs = [json.loads(line) for line in fp]
    dfs = []
    objs = []
    for i, rec in enumerate(tqdm.tqdm(user_recs)):
        objs += _process_user_record(rec)
        # create the dataframe in batches to reduce memory footprint
        if len(objs) >= 100000 or i == len(user_recs) - 1:
            batch_df = pd.io.json.json_normalize(objs)
            batch_df.day = pd.to_datetime(batch_df.day)
            batch_df.started = pd.to_datetime(batch_df.started, errors='coerce')
            dfs.append(batch_df)
            objs = []
    df = pd.concat(dfs)

    if pickled:
        logging.info(f"saving pickled user histories to {pickled}")
        df.to_pickle(pickled)

    return df


def _process_user_record(user: Dict[str, Any]) -> List[Dict[str, Any]]:
    objs = []

    for day, summary in user['days'].items():
        running_editors = 0
        vacancies = 0
        any_vacancy = False
        any_editor = False

        for editor in EDITORS:
            if summary[editor + '_running'] > 0:
                running_editors += 1
                any_editor = True
            if summary[editor + '_vacancy'] > 0:
                vacancies += 1
                any_vacancy = True

        obj = {
            'total_days': 1,

            'user_id': user['user_id'],
            'install_id': user['install_id'],
            'email': user['email'],
            'metrics_disabled': user['metrics_disabled'],
            'started': user['user_created_at'],
            'channel': user['channel'],
            'timezone': user['timezone'],
            'os': _max_key(user['os']),
            'memory_total': int(_max_key(user['memory_total'], default="0")),
            'cpu_mhz': float(_max_key(user['cpu_mhz'], default="0", include_fn=lambda k: float(k) != 0)),
            'last_surveyed_time': user['last_surveyed_time'],
            'last_surveyed_profession': user['last_surveyed_profession'],
            'last_surveyed_experience': user['last_surveyed_experience'],
            'last_surveyed_project_types': user['last_surveyed_project_types'],
            'last_surveyed_company_size': user['last_surveyed_company_size'],
            'last_seen': user['last_seen'],
            'last_login': user['last_login'],

            'day': day,

            'total_events': summary['total_events'],
            'python_events': summary['python_events'],
            'python_work_hours': _python_events_in_work_hours(pd.Timestamp(day), summary),
            'python_weekdays': _python_events_in_weekdays(pd.Timestamp(day), summary),
            'completions_shown': summary['completions_shown'],
            'active': int(summary['python_events'] >= 1),
            'very_active': int(summary['python_events'] >= 5),
            'any_vacancy': int(any_vacancy),
            'running_editors': running_editors,
            'any_editor': int(any_editor),
            'vacancies': vacancies,
            'git_found': summary['git_found'],
            'has_repo': summary['has_repo'],
            'time_zone_offset': int(_max_key(summary['time_zone_offsets'], default="0")),
        }

        for editor in EDITORS:
            obj[editor + '_plugin_installed'] = summary[editor + '_plugin_installed']
            obj[editor + '_installed'] = summary[editor + '_installed']
            obj[editor + '_running'] = summary[editor + '_running']

        objs.append(obj)

    return objs


def _convert_to_local(ts: pd.Timestamp, summary: Dict[str, Any]) -> pd.Timestamp:
    offset = int(_max_key(summary['time_zone_offsets'], default="0"))
    return ts + datetime.timedelta(seconds=offset)


def _python_events_in_work_hours(day: pd.Timestamp,
                                 summary: Dict[str, Any],
                                 work_start: int = 9, work_end: int = 17) -> int:
    total = 0
    for hr, events in enumerate(summary['python_events_by_hour']):
        ts = _convert_to_local(day + datetime.timedelta(hours=hr), summary)
        if work_start <= ts.hour < work_end:
            total += events

    return total


def _python_events_in_weekdays(day: pd.Timestamp, summary: Dict[str, Any]) -> int:
    total = 0
    for hr, events in enumerate(summary['python_events_by_hour']):
        ts = _convert_to_local(day + datetime.timedelta(hours=hr), summary)
        if 0 <= ts.weekday() < 5:
            total += events

    return total


def _max_key(d: Dict[str, int], default="", include_fn=None) -> str:
    if include_fn:
        d = {k: v for k, v in d.items() if include_fn(k)}
    if not d:
        return default
    return sorted(d.items(), key=lambda i: -i[1])[0][0]