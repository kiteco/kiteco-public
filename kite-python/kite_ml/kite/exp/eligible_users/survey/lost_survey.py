from typing import Any, Dict, List, Optional

import csv
import pandas as pd


def get_survey(survey_file: str, fix_file: Optional[str] = None) -> pd.DataFrame:
    """
    :param survey_file: The original survey csv (as exported by Typeform)
        see https://kite.quip.com/QwBYA04DuxfS/Lost-User-Survey
    :param fix_file: A csv of 30-ish users who answered questions at the beginning; these did not have user IDs
    :return: a DataFrame containing the parsed survey responses
    """

    recs = _parse_survey(survey_file, True)

    if fix_file is None:
        return pd.io.json.json_normalize(recs)

    fixed_recs = _parse_survey(fix_file, False)

    fix_ids = {rec['typeform_id']: rec['user_id'] for rec in fixed_recs}

    for rec in recs:
        typeform_id = rec['typeform_id']
        if typeform_id in fix_ids:
            rec['user_id'] = fix_ids[typeform_id]

    return pd.io.json.json_normalize(recs)


def _parse_survey(filename: str, use_kite_question: bool) -> List[Dict[str, Any]]:
    recs = []

    with open(filename) as csvfile:
        reader = csv.reader(csvfile)
        for i, row in enumerate(reader):
            if i == 0:
                continue  # skip header

            use_kite = ''

            if use_kite_question:
                (typeform_id, use_kite, last_used_py, primary_py_use, editor, other_editor,
                 supported_editor, feedback, _, uid, started, submitted, _) = row
            else:
                (typeform_id, last_used_py, primary_py_use, editor, other_editor,
                 supported_editor, feedback, _, uid, started, submitted, _) = row

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
                'use_kite': use_kite,
                'user_id': uid,
                'last_used_py': last_used_py,
                'primary_py_use': primary_py_use,
                'editor': editor,
                'eligible': int(eligible),
                'supported_editor': supported_editor,
                'feedback': feedback,
                'started': pd.Timestamp(started),
                'submitted': pd.Timestamp(submitted),
            })

    return recs
