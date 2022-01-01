from airflow import DAG
import datetime
from airflow.hooks.S3_hook import S3Hook
from elasticsearch import Elasticsearch
import json
import gzip
import io
import logging
import base64
from elasticsearch.helpers import bulk
from airflow.operators.python_operator import PythonOperator
from airflow.models import Variable
from airflow.contrib.operators.s3_list_operator import S3ListOperator
from airflow.models.xcom import XCom
import itertools
from airflow.operators.python_operator import ShortCircuitOperator
from jinja2 import PackageLoader
import time
from kite_airflow.plugins.google import GoogleSheetsRangeOperator
import kite_metrics
from kite_airflow.slack_alerts import task_fail_slack_alert


logger = logging.getLogger(__name__)

INDEX_GRANULARITY = datetime.timedelta(days=10)
BUCKET = 'kite-metrics'
KS_INDEX_PREFIX = 'kite_status'


def resolve_dotted_path(doc, path):
    container = doc
    field_name = path
    while '.' in field_name:
        container_name, field_name = path.split('.', 1)
        if container_name not in container:
            return None, None
        container = container[container_name]

    if field_name in container:
        return container, field_name

    return None, None


def get_index_shard(dt, granularity, epoch=datetime.date(1970, 1, 1)):
    date = datetime.date(dt.year, dt.month, dt.day)
    rounded = epoch + (date - epoch) // granularity * granularity
    return rounded.isoformat()


def iter_s3_file(s3_hook, bucket, key):
    json_file = s3_hook.get_key(key, BUCKET)
    for line in gzip.open(json_file.get()['Body']):
        yield json.loads(line)


def client_event_convert_fn(docs, index_date_suffix, deployments):
    for doc in docs:
        if 'messageId' not in doc:
            continue

        if 'properties' not in doc:
            continue

        event = doc.get('event')
        if event == 'Index Build':
            index_prefix = 'index_build'
        elif event == 'Completion Stats':
            index_prefix = 'completions_selected'
        else:
            continue

        index_name = '{}_{}'.format(index_prefix, index_date_suffix)

        for field in ['originalTimestamp']:
            if field in doc:
                del doc[field]

        for field in ['repo_stats', 'receivedAt', 'sentAt', 'sent_at', 'parse_info.parse_errors']:
            container, field_name = resolve_dotted_path(doc['properties'], field)
            if container:
                del container[field_name]

        for field in ['cpu_info.sum', 'lexical_metrics.score']:
            container, field_name = resolve_dotted_path(doc['properties'], field)
            if container:
                container[field_name] = float(container[field_name])

        for field in ['completion_stats']:
            if field in doc['properties']:
                # completions_stats is an encoded list
                data = doc['properties'][field]
                data = base64.b64decode(data)
                data = gzip.GzipFile(fileobj=io.BytesIO(data)).read()
                data = json.loads(data)
                del doc['properties'][field]
                # create one document per completion stat
                i = 0
                for stat in data:
                    i += 1
                    elem = doc
                    for key in stat:
                        elem['properties'][key] = stat[key]
                    yield {
                        '_index': index_name,
                        '_id': doc['messageId'] + "-" + str(i),
                        '_source': elem
                    }

            else:
                yield {
                    '_index': index_name,
                    '_id': doc['messageId'],
                    '_source': doc
                }


def scrub(a_dict, schema):
    res = {}
    for k, v in schema['properties'].items():
        if k not in a_dict:
            continue
        a_val = a_dict[k]
        elastic = v.get('elastic', False)
        if isinstance(a_val, dict):
            if elastic:
                res[k] = {k1: v1 for k1, v1 in a_val.items() if k1}
            elif 'properties' in v:
                res[k] = scrub(a_val, v)
            continue

        if elastic:
            res[k] = a_val

    return res


kite_status_config = kite_metrics.load_context('kite_status')
kite_status_schema = kite_metrics.load_schema('kite_status')


def kite_status_convert_fn(docs, index_date_suffix, deployments):
    total_time = 0
    for i, doc in enumerate(docs):
        if i and i % 10000 == 0:
            logger.info('Done {} records, avg time / record={}'.format(i, total_time / i))
        start_time = time.perf_counter()
        if doc.get('event') != 'kite_status':
            total_time += (time.perf_counter() - start_time)
            continue

        if not doc.get('messageId'):
            total_time += (time.perf_counter() - start_time)
            continue

        if 'properties' not in doc:
            total_time += (time.perf_counter() - start_time)
            continue

        if sum(doc['properties'].get('{}_events'.format(lang), 0) for lang in kite_status_config['languages']) == 0:
            total_time += (time.perf_counter() - start_time)
            continue

        index_name = '{}_active_{}'.format(KS_INDEX_PREFIX, index_date_suffix)

        doc = scrub(doc, kite_status_schema)

        for field in ['cpu_samples_list', 'active_cpu_samples_list']:
            if not doc['properties'].get(field):
                continue
            p = field.split('_')[:-2]
            new_field = '_'.join(['max'] + p)
            doc['properties'][new_field] = max(map(float, doc['properties'][field]))

        # We got some bogus timestamps, TODO: validate and cleanup data
        for field in ['license_expire', 'plan_end']:
            if isinstance(doc['properties'].get(field), int):
                if 0 < doc['properties'][field] < 2524636800:
                    doc['properties'][field] = datetime.datetime.fromtimestamp(doc['properties'][field])
                else:
                    del doc['properties'][field]

        # Next block is for backcompatibilty only
        # can be removed once the content of the PR https://github.com/kiteco/kiteco/pull/10638/ has been released to
        # most of our users
        for field in ['cpu_samples', 'active_cpu_samples']:
            if field in doc['properties']:
                samples_str = doc['properties'].pop(field)
                if len(samples_str) == 0:
                    continue
                p = field.split('_')[:-1]
                new_field = '_'.join(['max'] + p)
                doc['properties'][new_field] = max(map(float, samples_str.split(',')))

        deployment_id = doc['properties'].get('server_deployment_id')
        if deployment_id and deployment_id in deployments:
            doc['properties']['server_deployment_name'] = deployments[deployment_id]

        doc['payload_size'] = len(doc)
        total_time += (time.perf_counter() - start_time)
        yield {'_index': index_name, '_id': doc['messageId'], '_source': doc}


kite_status_dag = DAG(
    'elastic_load_kite_status',
    description='Load kite_status to Kibana.',
    default_args={
        'retries': 1,
        'retry_delay': datetime.timedelta(minutes=5),
        'start_date': datetime.datetime(2020, 10, 15),
        'on_failure_callback': task_fail_slack_alert,
    },
    schedule_interval='*/10 * * * *',
    jinja_environment_kwargs={
        'loader': PackageLoader('kite_airflow', 'templates')
    },
)

client_events_dag = DAG(
    'elastic_load_client_events',
    description='Load client_events to Kibana.',
    default_args={
        'retries': 1,
        'retry_delay': datetime.timedelta(minutes=5),
        'start_date': datetime.datetime(2020, 10, 15),
        'on_failure_callback': task_fail_slack_alert,
    },
    schedule_interval='*/10 * * * *',
    jinja_environment_kwargs={
        'loader': PackageLoader('kite_airflow', 'templates')
    },
)

convert_fns = {'kite_status': kite_status_convert_fn, 'client_events': client_event_convert_fn}


def bulk_index_metrics(bucket, s3_keys, granularity, key, deployments):
    s3_hook = S3Hook('aws_us_east_1')
    es = Elasticsearch(
        cloud_id="metrics:XXXXXXX",
        http_auth=("elastic",  Variable.get('elastic_password')),
    )

    def iter():
        for s3_key in s3_keys:
            dt = datetime.date(*map(int, s3_key.split('/')[2:5]))
            index_date_suffix = get_index_shard(dt, granularity)

            for rec in convert_fns[key](iter_s3_file(s3_hook, bucket, s3_key), index_date_suffix, deployments):
                yield rec

    bulk(es, iter())


def skip_no_new_files(ti, **kwargs):
    prev_files = set(itertools.chain(*[result.value for result in XCom.get_many(
        execution_date=ti.execution_date,
        dag_ids=ti.dag_id,
        task_ids=ti.task_id,
        include_prior_dates=True,
        limit=100
    )]))

    all_files = set(ti.xcom_pull(task_ids='list_prev_json_files') + (ti.xcom_pull(task_ids='list_next_json_files') or []))
    curr_files = list(all_files - prev_files)
    ti.xcom_push(key='curr_files', value=curr_files)
    return len(curr_files) > 0


for key, dag in [('kite_status', kite_status_dag), ('client_events', client_events_dag)]:
    list_ops = [
        S3ListOperator(
            aws_conn_id='aws_us_east_1',
            task_id='list_{}_json_files'.format(k),
            bucket='kite-metrics',
            prefix="firehose/{}/{{{{ (execution_date + macros.timedelta(hours={})).format('%Y/%m/%d/%H') }}}}/".format(key, diff),
            delimiter='/',
            dag=dag,
        ) for k, diff in [('prev', 0), ('next', 1)]
    ]

    def load_fn(ti, params, **kwargs):
        s3_keys = ti.xcom_pull(task_ids=skip_no_new_files.__name__, key='curr_files')
        logger.info("Loading files {}".format(', '.join(s3_keys)))

        deployments_data = ti.xcom_pull(task_ids='copy_server_deployments')['values']
        id_col = deployments_data[1].index('Deployment ID')
        name_col = deployments_data[1].index('Name')

        deployments = {d[id_col]: d[name_col] for d in deployments_data[2:] if len(d) > max(id_col, name_col) and d[name_col].strip()}
        bulk_index_metrics(BUCKET, s3_keys, INDEX_GRANULARITY, params['key'], deployments)
        return s3_keys

    list_ops >> ShortCircuitOperator(
        task_id=skip_no_new_files.__name__,
        python_callable=skip_no_new_files,
        dag=dag,
        provide_context=True,
        depends_on_past=True,
    ) >> GoogleSheetsRangeOperator(
        gcp_conn_id='google_cloud_kite_dev',
        spreadsheet_id='1-XXXXXXX',
        range='A:D',
        task_id='copy_server_deployments',
        dag=dag,
        provide_context=True,
    ) >> PythonOperator(
        python_callable=load_fn,
        task_id='load_{}'.format(key),
        dag=dag,
        provide_context=True,
        params={'key': key}
    )
