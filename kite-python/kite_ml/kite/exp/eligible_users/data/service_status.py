import json
import pandas as pd
import tqdm


def get_service_status(filename: str, users: pd.DataFrame) -> pd.DataFrame:
    """
    :param filename: the service status JSON records, as returned by the get-service-status pipeline
    :param users: the users DataFrame, as returned by data.get_user_totals()
    :return: a DataFrame of ServiceStatus events, indexed by the timestamp of the event
    """
    uids_by_install = {}
    for user_id, row in users.iterrows():
        install_id = row.install_id
        if install_id == '':
            install_id = user_id
        uids_by_install[install_id] = user_id

    fp = open(filename, 'r')
    lines = fp.readlines()
    objs = []
    for line in tqdm.tqdm(lines):
        rec = json.loads(line)
        if rec['kite_service_version'] == '':
            continue
        max_lifetime = 0
        if rec['kited_lifetimes_in_millis']:
            max_lifetime = max(rec['kited_lifetimes_in_millis'])

        install_id = rec['install_id']
        user_id = uids_by_install.get(install_id, '')

        obj = {
            'timestamp': rec['timestamp'],
            'install_id': install_id,
            'user_id': user_id,
            'kited_running': int(rec['num_kited_processes'] > 0),
            'kite_proxy': int(rec['send_to_kite_dot_com_not_segment']),
            'client_proxy': int(rec['allow_proxy']),
            'max_lifetime': max_lifetime,
        }
        objs.append(obj)

    df = pd.io.json.json_normalize(objs)
    df['timestamp'] = pd.to_datetime(df.timestamp)
    return df
