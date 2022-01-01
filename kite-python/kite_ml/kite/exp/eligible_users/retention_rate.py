from typing import Callable

import datetime
import pandas as pd


def naive_retention_fn(df: pd.DataFrame) -> pd.Series:
    """
    Calculates naive retention rate based entirely on the kite_status data

    :param df: a DataFrame containing user counts for the following columns
        - [unactivated, activate, lost, dormant] - e.g. as returned by counts_for_daily_cohort()
    :return: a Series, with the same index as df, containing the calculated retention rate for each row
    """
    return df.active / (df.active + df.lost + df.dormant)


def true_retention_fn(
        win_survey: pd.DataFrame,
        active_days: int) -> Callable[[pd.DataFrame], pd.Series]:
    """
    Returns a function that can calculate the "true retention" rate given a DataFrame of daily user counts by cohort
       (e.g. as returned by counts_for_daily_cohort())

    For the motivation behind finding the true retention rate, see:
        https://kite.quip.com/TJxqAJs7vz05/Eligible-Users
    ...but the gist of it is that we want to find out the rate at which eligible users (i.e. those who are Python
    coders who use a Kite-supported editor) retain.

    To achieve this, we begin with the assumption that when looking at just the numbers from kite_status events,
    our categorization of dormant and lost users may be incorrect (due to the fact that it may count ineligible users,
    or issues with collecting metrics).

    To fix this, we look at the results of the Windows user survey:
        https://kite.quip.com/DgfvAoP0WOma/Windows-User-Survey

    This survey includes questions about the last time a user coded in Python and the last time a user coded using
    Kite. From these we can determine whether a user is truly lost or dormant, at least if we take that user's survey
    responses at face value.

    We then calculate the fractions of:
    - lost (categorized by us) respondents who claim to have used Python but not Kite within the past <active_days>
        (lost_lost_rate)
    - lost (categorized by us) survey respondents who claim to have used both Kite and Python within the past
        <active_days> (lost_active_rate)
    - dormant (categorized by us) survey respondents who claim to have used Python but not Kite within the past
        <active_days> (dormant_lost_rate)
    - dormant (categorized by us) survey respondents who claim to have used Python but not Kite within the past
        <active_days> (dormant_active_rate)

    We then apply corrections to the retention rate by redistributing our lost and dormant users according to these
    calculated rates, using the assumption that this rate holds for every measured cohort.

    active_users =
        active_count + (dormant_count * dormant_active_rate) + (lost_count * lost_active_rate)

    churned_users =
        (dormant_count * dormant_lost_rate) + (lost_count * lost_lost_rate)

    true_retention_rate = active_users / (active_users + churned_users)

    :param histories: user history DataFrame, as returned by load_user_histories()
    :param users: users DataFrame, as returned by get_user_totals()
    :param win_survey: windows survey result, as returned by windows_survey.get_responses()
    :param active_days: the active-day definition (the "n" in "n-day active")
    :return: a function that operates on a DataFrame containing user counts for the following columns
        - [unactivated, activate, lost, dormant] and returns a series containing just one column with the retention rate
    """
    # determine what are the responses to the "last used Kite" / "last used Python" questions that indicate the user
    # is still using Kite/Python within the desired window
    if active_days == 14:
        active_choices = {'14'}
    elif active_days == 30:
        active_choices = {'14', '30'}
    elif active_days == 90:
        active_choices = {'14', '30', '90'}
    else:
        raise ValueError("active_days needs to be in {14,30,90}")

    # we only consider respondents who answered both of the "last used Kite"/ "last used Python" questions
    resps = win_survey[(win_survey.last_used_kite != '') & (win_survey.last_used_py != '')]

    lost_resps = resps[resps.last_day < resps.started - datetime.timedelta(days=active_days)]
    lost_active = lost_resps[
        lost_resps.last_used_kite.isin(active_choices) & lost_resps.last_used_py.isin(active_choices)]
    lost_active_rate = len(lost_active) / len(lost_resps)
    lost_lost = lost_resps[
        ~lost_resps.last_used_kite.isin(active_choices) & lost_resps.last_used_py.isin(active_choices)]
    lost_lost_rate = len(lost_lost) / len(lost_resps)

    dormant_resps = resps[(resps.last_day >= resps.started - datetime.timedelta(days=active_days)) &
                          (resps.last_py_event < resps.started - datetime.timedelta(days=active_days))]
    dormant_active = dormant_resps[
        dormant_resps.last_used_kite.isin(active_choices) & dormant_resps.last_used_py.isin(active_choices)]
    dormant_active_rate = len(dormant_active) / len(dormant_resps)
    dormant_lost = dormant_resps[
        ~dormant_resps.last_used_kite.isin(active_choices) & dormant_resps.last_used_py.isin(active_choices)]
    dormant_lost_rate = len(dormant_lost) / len(dormant_resps)

    def retention_fn(df: pd.DataFrame) -> pd.Series:
        active = df.active + (df.dormant * dormant_active_rate) + (df.lost * lost_active_rate)
        churned = (df.dormant * dormant_lost_rate) + (df.lost * lost_lost_rate)
        return active / (active + churned)

    return retention_fn



