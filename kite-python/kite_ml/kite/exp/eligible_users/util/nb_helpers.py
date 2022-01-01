from typing import Dict, Callable

import pandas as pd


def split_by(groups: Dict[str, pd.DataFrame], row_fn: Callable[[pd.Series], str]):
    d = {}
    for label in sorted(groups.keys()):
        users = groups[label].copy()
        users['sel'] = row_fn(users)
        grouped = users.groupby(['sel']).size()
        d[label] = grouped
        d[label + ' %'] = grouped / grouped.sum() * 100

    return pd.DataFrame(d)

