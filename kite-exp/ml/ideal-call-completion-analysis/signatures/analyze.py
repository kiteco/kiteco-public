from typing import NamedTuple, List, Dict

from kite.asserts.asserts import FieldValidator


import datetime
import json

import numpy as np


class SignatureData(NamedTuple):
    user_id: str
    sent_at: datetime.datetime
    triggered: int
    shown: int

    @classmethod
    def from_json(cls, d: dict) -> 'SignatureData':
        v = FieldValidator(cls, d)
        return SignatureData(
            user_id=v.get('user_id', str),
            sent_at=datetime.datetime.fromtimestamp(v.get('sent_at', int)),
            triggered=v.get('signatures_triggered', int),
            shown=v.get('signatures_shown', int),
        )


def read_data(fname: str, num=100) -> List[SignatureData]:
    data = []
    with open(fname) as f:
        for line in f:
            if len(data) >= num > 0:
                return data
            data.append(SignatureData.from_json(json.loads(line)))
    return data


def by_user(data: List[SignatureData]) -> Dict[str, List[SignatureData]]:
    users = {}
    for d in data:
        if d.user_id not in users:
            users[d.user_id] = []
        users[d.user_id].append(d)
    return users


def by_day(data: List[SignatureData]) -> Dict[str, List[SignatureData]]:
    days = {}
    for d in data:
        ts = '{}:{}'.format(d.sent_at.month, d.sent_at.day)
        if ts not in days:
            days[ts] = []
        days[ts].append(d)
    return days


ByUserByDay = Dict[str, Dict[str, List[SignatureData]]]


def by_user_by_day(data: List[SignatureData]) -> ByUserByDay:
    user_by_day = {}
    for usr, ds in by_user(data).items():
        user_by_day[usr] = by_day(ds)
    return user_by_day


def filter_inactive_days(bubd: ByUserByDay, min_active_events: int) -> ByUserByDay:
    new_bubd = {}
    for usr, days in bubd.items():
        new_days = {}
        for day, evts in days.items():
            active_count = 0
            for evt in evts:
                if evt.triggered > 0:
                    active_count += 1
            if active_count >= min_active_events:
                new_days[day] = evts
        if len(new_days) > 0:
            new_bubd[usr] = new_days
    return new_bubd


percentiles = [25, 50, 75, 95]


def percentiles_str(name: str, data: list) -> str:
    return '{} percentiles ({}): {}'.format(name, percentiles, np.percentile(data, percentiles))


def percentiles_triggered_per_day(bubd: ByUserByDay):
    per_days = []
    for _, days in bubd.items():
        for _, day in days.items():
            per_day = 0
            for evt in day:
               per_day += evt.triggered
            per_days.append(per_day)
    print(percentiles_str('Triggered per day', per_days))


def percentiles_shown_per_day(bubd: ByUserByDay):
    per_days = []
    for _, days in bubd.items():
        for _, day in days.items():
            per_day = 0
            for evt in day:
                per_day += evt.shown
            per_days.append(per_day)
    print(percentiles_str('Shown per day', per_days))


if __name__ == '__main__':
    data = read_data('data.json', 0)
    bubd = by_user_by_day(data)
    bubd = filter_inactive_days(bubd, 1)
    print('after filtering got', len(bubd))
    percentiles_triggered_per_day(bubd)
    percentiles_shown_per_day(bubd)
