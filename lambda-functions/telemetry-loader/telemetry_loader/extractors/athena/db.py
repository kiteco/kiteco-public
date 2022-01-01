import boto3
import datetime
import logging
import hashlib
import os
import pystache
import pytz
import time

logger = logging.getLogger(__name__)

SQL_FILE = os.path.join(os.path.dirname(os.path.realpath(__file__)), 'kite_status_1d.sql.tmpl')
TIMEZONE = 'UTC'


def dt_to_utc(dt):
    # is_dst is only used at DST switch-overs. It's usually ignored.
    return pytz.timezone(TIMEZONE).localize(dt, is_dst=False).astimezone(pytz.UTC)


DATA_LOC = 's3://kite-metrics/firehose/kite_status/'
PROD_RESULT_LOC_PREFIX = 's3://kite-metrics/athena-results'
TEST_RESULT_LOC_PREFIX = 's3://kite-metrics/athena-results-test'


class Queries(object):
    kite_status_1d = ('kite_status', 'kite_status_table.sql.tmpl', 'kite_status_1d', 'kite_status_1d.sql.tmpl')


def run_query(config, start, end, prod=True):
    client = boto3.client('athena')
    table_name_prefix, create_table_tmpl, query_name, query_tmpl = config
    result_loc_prefix = PROD_RESULT_LOC_PREFIX if prod else TEST_RESULT_LOC_PREFIX

    create_table_sql, table_name = _sql_create_versioned_table(table_name_prefix, create_table_tmpl)
    wait(client, _execute_query(client, create_table_sql, result_loc_prefix, 'ddl'))

    utc_time_range = [dt_to_utc(dt) for dt in [start, end]]

    logger.info("Verifying partitions...")

    partition_sql = _get_partition_sql(table_name, DATA_LOC, utc_time_range)

    responses = _execute_query(client, partition_sql, result_loc_prefix, 'ddl')

    for i, resp in enumerate(responses):
        wait(client, resp)
        logger.info('    {} / {} partitions created.'.format(i + 1, len(responses)))

    data_query = _get_data_query_sql(table_name, query_tmpl, TIMEZONE, utc_time_range)
    _execute_query(client, data_query, result_loc_prefix, query_name)


def _sql_create_versioned_table(prefix, create_table_template):
    with open(os.path.join(os.path.dirname(__file__), create_table_template), 'rb') as file:
        template_content = file.read()

    h = hashlib.sha1(template_content)
    ctx = {'table_name': '{}_{}'.format(prefix, h.hexdigest())}

    return pystache.render(template_content, ctx), 'kite_metrics.{}'.format(ctx['table_name'])


def _format_prefix_string(dt):
    return dt.strftime('%Y/%m/%d/%H')


def _get_partition_sql(table_name, location_prefix, time_range):
    start, end = get_prefix_bounds(*time_range)

    partitions = []

    while start <= end:
        partitions.append(_format_prefix_string(start))
        start += datetime.timedelta(hours=1)

    return ['ALTER TABLE {table} ADD IF NOT EXISTS PARTITION (prefix=\'{partition}\') LOCATION \'{prefix}{partition}/\';'.format(
            table=table_name, prefix=location_prefix, partition=partition) for partition in partitions]


def _get_data_query_sql(table_name, query_template, timezone, time_range):
    timestamps = [format_timestamp_string(dt) for dt in time_range]
    prefix_bounds_str = [_format_prefix_string(dt) for dt in get_prefix_bounds(*time_range)]

    with open(os.path.join(os.path.dirname(__file__), query_template), 'r') as template:
        return pystache.render(template.read(), {
            'table_name': table_name, 'interval': 'day', 'tz': TIMEZONE,
            'start_prefix': prefix_bounds_str[0], 'end_prefix': prefix_bounds_str[1],
            'start_timestamp': timestamps[0], 'end_timestamp': timestamps[1]})


def _execute_query(client, query_sql, output_prefix, output_suffix):
    if isinstance(query_sql, (list, tuple)):
        return [_execute_query(client, qs, output_prefix, output_suffix) for qs in query_sql]

    return client.start_query_execution(
        QueryString=query_sql,
        QueryExecutionContext={'Database': 'kite_metrics'},
        ResultConfiguration={'OutputLocation': '/'.join([output_prefix, output_suffix])})


def wait(client, query, timeout=600):
    start = time.time()

    while time.time() - start < timeout:
        response = client.get_query_execution(QueryExecutionId=query['QueryExecutionId'])

        if 'QueryExecution' in response and \
                'Status' in response['QueryExecution'] and \
                'State' in response['QueryExecution']['Status']:
            state = response.get('QueryExecution', {}).get('Status', {}).get('State')
            if state == 'FAILED':
                return response
            if state == 'SUCCEEDED':
                return response
        time.sleep(5)
    return False


def get_prefix_bounds(start, end, buffer=60 * 15):
    return ((start - datetime.timedelta(seconds=buffer)).replace(minute=0, second=0, microsecond=0),
            (end + datetime.timedelta(hours=1, seconds=buffer)).replace(minute=0, second=0, microsecond=0))


def format_timestamp_string(dt):
    return dt.strftime('%Y-%m-%dT%H:%M:%S')
