from typing import Set

import pandas as pd


EDITORS = ['atom', 'intellij', 'sublime3', 'vim', 'vscode']


# TODO: what uses this?
def get_dormant_users_by_vacancy(
        histories: pd.DataFrame,
        user_ids: Set[str],
        have_vacancies: bool,
        consecutive_dormant: int = 4) -> Set[int]:
    grouped = histories[histories.user_id.isin(user_ids)].groupby(['user_id', 'day']).sum()

    uids = set([])
    consecutive = 0
    last_user = None

    for idx, rec in grouped.iterrows():
        user_id, day = idx

        if user_id != last_user:
            consecutive = 0
            last_user = user_id

        if have_vacancies:
            vac_match = (rec.any_vacancy > 0)
        else:
            vac_match = (rec.any_vacancy == 0)

        if rec.python_events == 0 and vac_match:
            consecutive += 1
        else:
            consecutive = 0

        if consecutive >= consecutive_dormant:
            uids.add(user_id)

    return uids


def get_first_day(histories: pd.DataFrame) -> pd.Timestamp:
    """
    get_first_day gets the first day of the histories DataFrame in a way that is robust to outliers - for some
    reason, a small number of user summaries are very far into the past or future
    """
    return histories.day.nsmallest(50).iloc[-1]


def get_last_day(histories: pd.DataFrame) -> pd.Timestamp:
    """
    get_last_day gets the last day of the histories DataFrame in a way that is robust to outliers - for some
    reason, a small number of user summaries are very far into the past or future
    """
    return histories.day.nlargest(50).iloc[-1]
