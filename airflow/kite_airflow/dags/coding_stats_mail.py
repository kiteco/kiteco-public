import concurrent.futures
import datetime
import threading

from airflow import DAG
from airflow.operators.python_operator import PythonOperator
from airflow.contrib.operators.aws_athena_operator import AWSAthenaOperator
from jinja2 import PackageLoader

from customerio import CustomerIO
from kite_airflow.common import configs
from kite_airflow.common import utils
from kite_airflow.common import files
from kite_airflow.slack_alerts import task_fail_slack_alert


DIR_BASE_URI = 's3://{}/{}'.format(configs.BUCKET, 'coding-stats-mail')
DIR_APPROX_PERCENTILES = 'approx_percentiles'
DIR_DAILY_ACTIVE_USERS = 'daily_active_users'
DIR_CODING_STATS = 'coding_stats'

TABLE_DAILY_ACTIVE_USERS = 'kite_daily_active_users' if utils.is_production() else 'kite_daily_active_users_dev'

USER_LIMIT = -1 # helpful to reduce time during development
NUM_OF_WEEKS = 6
EVENT_STATS_EMAIL = 'send_stats_email_weekly'

cio_local = threading.local()

default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime.datetime(2021, 1, 24),
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 0,
    'retry_delay': datetime.timedelta(minutes=5),
    'on_failure_callback': task_fail_slack_alert,
}

kite_coding_stats_email_dag = DAG(
    'kite_coding_stats_mail',
    description='Weekly coding stats emails to users that are active in last 2 weeks.',
    default_args=default_args,
    schedule_interval='0 20 * * SUN', # Every Sunday 20:00
    jinja_environment_kwargs={
        'loader': PackageLoader('kite_airflow', 'templates')
    },
)

approx_percentiles_op = AWSAthenaOperator(
    aws_conn_id=configs.AWS_CONN_ID,
    task_id='get_approx_percentiles',
    query='athena/coding_stats_mail/queries/approx_percentiles.sql',
    params={
        'languages': utils.get_supported_languages(),
    },
    output_location='{}/{}/'.format(configs.DIR_SCRATCH_URI, DIR_APPROX_PERCENTILES),
    database=configs.DB_KITE_METRICS,
    dag=kite_coding_stats_email_dag,
)

drop_daily_active_users_op = AWSAthenaOperator(
    aws_conn_id=configs.AWS_CONN_ID,
    task_id='drop_daily_active_users',
    query='athena/coding_stats_mail/queries/drop_daily_active_users.sql',
    params={
        'table_name': TABLE_DAILY_ACTIVE_USERS,
    },
    output_location='{}/{}/'.format(configs.DIR_SCRATCH_URI, DIR_DAILY_ACTIVE_USERS),
    database=configs.DB_KITE_METRICS,
    dag=kite_coding_stats_email_dag,
)

create_daily_active_users_op = AWSAthenaOperator(
    aws_conn_id=configs.AWS_CONN_ID,
    task_id='create_daily_active_users',
    query='athena/coding_stats_mail/tables/kite_daily_active_users.sql',
    params={
        'table_name': TABLE_DAILY_ACTIVE_USERS,
        'data_location': '{}/{}/'.format(DIR_BASE_URI, DIR_DAILY_ACTIVE_USERS),
    },
    output_location='{}/{}/'.format(configs.DIR_SCRATCH_URI, DIR_DAILY_ACTIVE_USERS),
    database=configs.DB_KITE_METRICS,
    dag=kite_coding_stats_email_dag,
)

update_daily_active_users_op = AWSAthenaOperator(
    aws_conn_id=configs.AWS_CONN_ID,
    task_id='update_daily_active_users',
    query='athena/coding_stats_mail/queries/update_daily_active_users.sql',
    params={
        'table_name': TABLE_DAILY_ACTIVE_USERS,
        'languages': utils.get_supported_languages(),
    },
    output_location='{}/{}/'.format(configs.DIR_SCRATCH_URI, DIR_DAILY_ACTIVE_USERS),
    database=configs.DB_KITE_METRICS,
    dag=kite_coding_stats_email_dag,
)

coding_stats_op = AWSAthenaOperator(
    aws_conn_id=configs.AWS_CONN_ID,
    task_id='coding_stats',
    query='athena/coding_stats_mail/queries/coding_stats.sql',
    params={
        'table_daily_active_users': TABLE_DAILY_ACTIVE_USERS,
        'languages': utils.get_supported_languages(),
        'num_of_weeks': NUM_OF_WEEKS,
    },
    output_location='{}/{}/'.format(configs.DIR_SCRATCH_URI, DIR_CODING_STATS),
    database=configs.DB_KITE_METRICS,
    dag=kite_coding_stats_email_dag,
)


def get_approx_percentiles(ti):
    percentiles_list = files.get_full_scratch_space_csv(
        ti,
        approx_percentiles_op.task_id,
        DIR_APPROX_PERCENTILES,
    )[0]

    approx_percentiles = []
    for percentile_index in range(1, 100):
        approx_percentiles.append({
            "percentile": percentile_index,
            "value": float(percentiles_list[f'pct_{percentile_index}']),
        })

    return approx_percentiles


def get_coding_time_percentile(coding_hours, percentiles):
    max_coding_time_percentile = 0

    for index in range(len(percentiles)):
        if percentiles[index]["value"] <= coding_hours:
            max_coding_time_percentile = percentiles[index]["percentile"]

    return max_coding_time_percentile


def is_inactive_user(record):
    '''
    Checks if user is inactive by looking at first two weeks of data

    Adding first two weeks of coding_hours & completions_selected, if both of them are zero
    then returns True
    '''
    return (
        (record['coding_hours'].get(0, 0) + record['coding_hours'].get(1, 0) == 0) and
        (record['completions_selected'].get(0, 0) + record['completions_selected'].get(1, 0) == 0)
    )


def get_track_object(coding_stat_row, execution_date, all_percentiles):
    '''Returns the track_object OR None in case of inactive user'''

    # transforms coding stat data to their respective types
    coding_stat_row['total_weeks'] = int(coding_stat_row['total_weeks'])
    coding_stat_row['streak'] = int(coding_stat_row['streak'])
    coding_stat_row['completions_selected'] = utils.string_to_dict(coding_stat_row['completions_selected'])
    coding_stat_row['coding_hours'] = utils.string_to_dict(coding_stat_row['coding_hours'])
    coding_stat_row['python_hours'] = utils.string_to_dict(coding_stat_row['python_hours'])

    if is_inactive_user(coding_stat_row):
        return None

    coding_time_graph = []
    max_coding_hours = max(coding_stat_row['coding_hours'].values())
    max_python_hours = max(coding_stat_row['python_hours'].values())

    exec_date_end = datetime.datetime(execution_date.year, execution_date.month, execution_date.day) + datetime.timedelta(days=7)
    sat_offset = (exec_date_end.weekday() - 5) % 7
    sun_offset = (exec_date_end.weekday() - 6) % 7

    for week_index in range(NUM_OF_WEEKS - 1, -1, -1):
        start_date = exec_date_end.replace(hour=0, minute=0, second=0) - datetime.timedelta(days=7 * (week_index + 1) + sun_offset) # Sunday 12:00am
        end_date = exec_date_end.replace(hour=23, minute=59, second=59) - datetime.timedelta(days=7 * week_index + sat_offset) # Saturday 11:59:59pm

        coding_hours = coding_stat_row['coding_hours'].get(week_index, 0)
        python_hours = coding_stat_row['python_hours'].get(week_index, 0)
        completions_selected = coding_stat_row['completions_selected'].get(week_index, 0)

        coding_time_graph.append({
            'start_date': int(start_date.timestamp()),
            'end_date': int(end_date.timestamp()),
            'coding_hours': coding_hours,
            'scaled_coding_hours': coding_hours / max_coding_hours if max_coding_hours > 0 else 0,
            'py_hours': python_hours,
            'scaled_py_hours': python_hours / max_python_hours if max_python_hours > 0 else 0,
            'completions_used': completions_selected,
            'time_saved': python_hours * 0.18,
        })

    return dict(
        all_time_weeks = coding_stat_row['total_weeks'],
        streak = coding_stat_row['streak'],
        coding_time_percentile = get_coding_time_percentile(
            coding_stat_row['coding_hours'].get(week_index, 0),
            all_percentiles
        ),
        coding_time_graph = coding_time_graph,
    )


def iteration(ti, execution_date, storage_task_name):
    all_percentiles = get_approx_percentiles(ti)
    start_row = ti.xcom_pull(task_ids=storage_task_name, key='progress')

    for i, coding_stat_row in files.get_line_of_scratch_space_csv(ti, coding_stats_op.task_id, DIR_CODING_STATS):
        if i <= start_row:
            continue

        yield (
            i,
            coding_stat_row['userid'],
            get_track_object(coding_stat_row, execution_date, all_percentiles)
        )

        if i == USER_LIMIT:
            return


def send_event_to_cio(item):
    i, userid, track_object = item

    if not hasattr(cio_local, 'client'):
        cio_local.client = CustomerIO(configs.CIO_CREDENTIALS['site_id'], configs.CIO_CREDENTIALS['api_key'])

    if track_object != None:
        cio_local.client.track(customer_id=userid, name=EVENT_STATS_EMAIL, **track_object)

    return i


def submissions_to_cio(ti, execution_date, dag_run, storage_task_name, **context):
    queue_size = 100
    futures = []
    records_iter = iteration(ti, execution_date, storage_task_name)
    has_values = True

    with concurrent.futures.ThreadPoolExecutor(max_workers=configs.CIO_MAX_CONCURRENT_REQUESTS) as executor:
        while has_values:
            while len(futures) < queue_size:
                try:
                    futures.append(executor.submit(send_event_to_cio, next(records_iter)))
                except StopIteration:
                    has_values = False
                    break

            mode = concurrent.futures.FIRST_COMPLETED if has_values else concurrent.futures.ALL_COMPLETED
            done, not_done = concurrent.futures.wait(futures, timeout=6000, return_when=mode)
            futures = list(not_done)

            for future in done:
                try:
                    i = future.result()

                except Exception:
                    dag_run.get_task_instance(storage_task_name).xcom_push(
                        key='progress',
                        value=i - configs.CIO_MAX_CONCURRENT_REQUESTS  # subtracting because due to threading we can't get the exact index
                    )
                    raise


progress_storage_operator = PythonOperator(
    python_callable=lambda ti, **kwargs: ti.xcom_push(key='progress', value=0),
    task_id='progress_storage_{}'.format(submissions_to_cio.__name__),
    dag=kite_coding_stats_email_dag,
    provide_context=True,
)

submissions_to_cio_operator = PythonOperator(
    python_callable=submissions_to_cio,
    task_id=submissions_to_cio.__name__,
    dag=kite_coding_stats_email_dag,
    provide_context=True,
    op_kwargs={'storage_task_name': 'progress_storage_{}'.format(submissions_to_cio.__name__)}
)

(
    approx_percentiles_op,
    drop_daily_active_users_op >> create_daily_active_users_op >> update_daily_active_users_op >> coding_stats_op,
    progress_storage_operator,
) >> submissions_to_cio_operator
