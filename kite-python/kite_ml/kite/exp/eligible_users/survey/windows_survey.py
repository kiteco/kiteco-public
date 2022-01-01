from typing import Any, Dict, List, Optional

import csv
import logging
import pandas as pd
import tqdm


def get_responses(responses_file: str, histories: pd.DataFrame) -> pd.DataFrame:
    """
    :param responses_file: The original survey CSV file (as exported by Typeform)
        see https://kite.quip.com/DgfvAoP0WOma/Windows-User-Survey
    :param histories: DataFrame for the users' histories, as returned by load_user_histories()
    :return: a DataFrame containing the parsed survey responses, along with relevant information for each user
        taken from our histories
    """
    logging.info(f"getting survey responses from {responses_file}")
    recs = _parse_responses(responses_file)
    resps = pd.io.json.json_normalize(recs).set_index('user_id')

    # for each respondent, we find the day of their first kite_status event, the day of their last kite_status event,
    # and the day of their last Python event (as of the time of their response)
    first_day = {}
    last_day = {}
    last_py_event = {}

    for uid, row in tqdm.tqdm(resps.iterrows(), total=len(resps)):
        hist = histories[histories.user_id == uid]
        hist = hist[hist.day <= row.started]

        first_day[uid] = hist.day.min()
        last_day[uid] = hist.day.max()
        last_py_event[uid] = hist[hist.python_events > 0].day.max()

    resps['first_day'] = pd.Series(first_day)
    resps['last_day'] = pd.Series(last_day)
    resps['last_py_event'] = pd.Series(last_py_event)

    return resps


def _parse_responses(responses_file: str) -> List[Dict[str, Any]]:
    recs = []

    with open(responses_file) as csvfile:
        reader = csv.reader(csvfile)
        for i, row in enumerate(reader):
            if i == 0:
                continue  # skip header

            (typeform_id,
             last_used_kite,
             recommend_kite,
             last_used_py,
             primary_py_use,
             editor,
             other_editor,
             supported_editor,
             feedback,
             willing_to_talk,
             uid,
             started,
             submitted,
             _) = row

            last_used_kite = {
                'the last 14 days': '14',
                'the last 30 days': '30',
                'the last 90 days': '90',
                'the last year': 'year',
                'I have never used Kite': 'never',
                '': '',
            }[last_used_kite]

            last_used_py = {
                'the last 14 days': '14',
                'the last 30 days': '30',
                'the last 90 days': '90',
                'the last year': 'year',
                'I have never coded in Python': 'never',
                'I stopped coding in Python': 'stopped',
                '': '',
            }[last_used_py]

            primary_py_use = {
                'Work': 'work',
                'Non-work activities, like leisure or school': 'non-work',
                '': '',
            }[primary_py_use]

            editor = {
                'PyCharm / IntelliJ': 'intellij',
                'VS Code': 'vscode',
                'Atom': 'atom',
                'Jupyter Notebooks or Lab': 'jupyter',
                'Sublime Text': 'sublime3',
                'Vim': 'vim',
                'Spyder': 'spyder',
                'Emacs': 'emacs',
                'Other': 'other',
                '': '',
            }[editor]
            editor_non_other = editor
            if other_editor:
                editor = other_editor

            eligible_editor = False
            if editor in {'intellij', 'vscode', 'atom', 'vim', 'sublime3'}:
                eligible_editor = True
            elif supported_editor == '1':
                eligible_editor = True

            eligible = eligible_editor and last_used_py in {'14', '30', '90'}

            recs.append({
                'typeform_id': typeform_id,
                'last_used_kite': last_used_kite,
                'recommend_kite': recommend_kite,
                'user_id': uid,
                'last_used_py': last_used_py,
                'primary_py_use': primary_py_use,
                'editor_non_other': editor_non_other,
                'editor': editor,
                'eligible': int(eligible),
                'supported_editor': supported_editor,
                'feedback': feedback,
                'started': pd.Timestamp(started, tz='utc'),
                'submitted': pd.Timestamp(submitted, tz='utc'),
                'willing_to_talk': willing_to_talk,
            })

    return recs
