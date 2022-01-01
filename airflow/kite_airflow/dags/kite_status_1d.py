from datetime import timedelta
import base64
import hashlib
import mixpanel
import gzip
import json
import customerio
from airflow.contrib.operators.s3_list_operator import S3ListOperator
# The DAG object; we'll need this to instantiate a DAG
from airflow import DAG
# Operators; we need this to operate!
from airflow.contrib.operators.aws_athena_operator import AWSAthenaOperator
from airflow.models import Variable
from airflow.hooks.S3_hook import S3Hook
from airflow.operators.python_operator import PythonOperator
from airflow.operators.python_operator import ShortCircuitOperator
import logging
import datetime
from jinja2 import PackageLoader
import kite_metrics
from kite_airflow.slack_alerts import task_fail_slack_alert


logger = logging.getLogger(__name__)
MP_START_DATE = datetime.datetime(2020, 5, 29)

default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime.datetime(2020, 5, 24),
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 0,
    'retry_delay': timedelta(minutes=5),
    'on_failure_callback': task_fail_slack_alert,
}


DATA_LOC = 's3://kite-metrics/firehose/kite_status/'
PROD_RESULT_LOC_PREFIX = 's3://kite-metrics/athena-results'

dag = DAG(
    'kite_status_1d',
    default_args=default_args,
    description='A simple tutorial DAG',
    schedule_interval='10 0 * * *',
    jinja_environment_kwargs={
        'loader': PackageLoader('kite_airflow', 'templates')
    },
)

kite_status_config = kite_metrics.load_context('kite_status')
kite_status_schema = kite_metrics.load_schema('kite_status')

schema_reload_ops = []

for table_name in ['kite_status', 'kite_status_segment', 'kite_status_normalized']:
    schema_reload_ops.append(AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='drop_{}'.format(table_name),
        query='DROP TABLE {{params.table_name}}',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        dag=dag,
        params={'table_name': table_name},
    ) >> AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='create_{}'.format(table_name),
        query='athena/tables/{}.tmpl.sql'.format(table_name),
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        dag=dag,
        params={'schema': kite_status_schema, 'table_name': table_name}
    ))

insert_kite_status_normalized = AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='insert_kite_status_normalized',
    query='athena/queries/kite_status_normalized.tmpl.sql',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
    params={'schema': kite_status_schema}
)

cleanup_kite_status_normalized_table = AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='cleanup_kite_status_normalized_table',
    query='DROP TABLE kite_status_normalized_{{ds_nodash}}',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
)

schema_reload_ops >> insert_kite_status_normalized >> cleanup_kite_status_normalized_table


def read_s3_json_files(bucket, file_list):
    s3 = S3Hook('aws_us_east_1')

    for file in sorted(file_list):
        obj = s3.get_key(file, bucket)
        for line in gzip.open(obj.get()['Body']):
            rec = json.loads(line)
            to_clean = [rec]
            while to_clean:
                this = to_clean.pop()
                for k in list(this.keys()):
                    v = this[k]
                    if isinstance(v, dict):
                        to_clean.append(v)
                        continue
                    if v is None:
                        del this[k]
            yield rec


def load_athena_to_elastic(task_instance, execution_date, **context):
    from elasticsearch import Elasticsearch
    from elasticsearch.helpers import bulk
    es = Elasticsearch(
        cloud_id="metrics:XXXXXXX",
        http_auth=("elastic", Variable.get('elastic_password')),
    )

    def iter():
        iter_records = read_s3_json_files('kite-metrics', task_instance.xcom_pull(task_ids='list_mixpanel_json_files'))
        for i, rec in enumerate(iter_records):
            try:
                if sum(rec.get('{}_events'.format(lang), 0) for lang in kite_status_config['languages']) == 0:
                    continue

                if rec['event'] != 'kite_status':
                    continue

                ts = datetime.datetime.fromtimestamp(rec['end_time'])

                rec_id_str = '{}::{}'.format(rec.get('userid', ''), ts.strftime('%Y/%m/%d'))
                rec_id = hashlib.md5(rec_id_str.encode('utf8')).hexdigest()
                rec['timestamp'] = ts
                yield {'_index': 'kite_status_1d_{}'.format(execution_date.format('%Y%m')), '_id': rec_id, '_source': rec}
            except Exception:
                logger.exception("Error processing line {}, content={}".format(i, rec))
                raise

    bulk(es, iter())


event_names = {
    'anon_supported_file_edited': 'anon_supported_file_edited_1d',
    'anon_kite_status': 'anon_kite_status_1d',
    'kite_status': 'kite_status_1d',
}


def load_athena_to_mixpanel(task_instance, execution_date, dag_run, storage_task_name, **context):
    mp_consumer = mixpanel.BufferedConsumer(max_size=100)
    mp_client = mixpanel.Mixpanel(Variable.get('mixpanel_credentials', deserialize_json=True)['token'], consumer=mp_consumer)
    start_row = task_instance.xcom_pull(task_ids=storage_task_name, key='progress')

    iter_records = read_s3_json_files('kite-metrics', task_instance.xcom_pull(task_ids='list_mixpanel_json_files'))
    for i, rec in enumerate(iter_records):
        if i <= start_row:
            continue
        try:
            insert_id = str(base64.b64encode(
                hashlib.md5('{}::{}'.format(
                    rec['userid'],
                    execution_date.strftime('%Y/%m/%d')).encode('utf8')
                ).digest())[:16])
            rec.update({
                'time': rec['end_time'],
                '_group': 'firehose/kite_status/{}/'.format(execution_date.strftime('%Y/%m/%d')),
                '_version': '1.0.0',
                '$insert_id': insert_id,
            })
            user_id = rec['userid']
            name = event_names.get(rec['event'])
            if name is None:
                continue

            if datetime.datetime.today() - execution_date < datetime.timedelta(days=4):
                mp_client.track(user_id, name, rec)
            else:
                ts = rec.pop('time')
                mp_client.import_data(Variable.get('mixpanel_credentials', deserialize_json=True)['api_key'], user_id, name, ts, rec)
            if i > 0 and i % 10000 == 0:
                logger.info("Processed line {}".format(i))
                dag_run.get_task_instance(storage_task_name).xcom_push(key='progress', value=i)
        except Exception:
            dag_run.get_task_instance(storage_task_name).xcom_push(key='progress', value=i-100)
            logger.exception("Error processing line {}, content={}".format(i, rec))
            raise
    mp_consumer.flush()


def load_athena_to_cio(task_instance, execution_date, dag_run, storage_task_name, **context):
    import concurrent.futures
    cio_creds = Variable.get('cio_credentials', deserialize_json=True)
    start_row = task_instance.xcom_pull(task_ids=storage_task_name, key='progress')
    iter_records = read_s3_json_files('kite-metrics', task_instance.xcom_pull(task_ids='list_cio_json_files'))

    def iter():
        for i, rec in enumerate(iter_records):

            if i <= start_row:
                continue

            if rec['event'] != 'kite_status':
                continue

            rec.update({
                'time': rec['end_time'],
                '_group': 'firehose/kite_status/{}/'.format(execution_date.strftime('%Y/%m/%d')),
                '_version': '1.0.0',
            })
            user_id = rec['userid']

            if not user_id or not all(ord(c) < 128 for c in user_id):
                continue

            name = event_names.get(rec['event'])
            if name is None:
                continue

            yield i, (user_id, name, rec['time']), rec

    def call_cio(item):
        i, args, kwargs = item
        customerio.CustomerIO(cio_creds['site_id'], cio_creds['api_key']).backfill(*args, **kwargs)
        return i

    max_i = 0
    with concurrent.futures.ThreadPoolExecutor(max_workers=20) as executor:
        try:
            for i in executor.map(call_cio, iter()):
                if max_i > 0 and (i // 1000) > (max_i // 1000):
                    logger.info("Processed line {}".format(i))
                    dag_run.get_task_instance(storage_task_name).xcom_push(key='progress', value=max(max_i, i))
                max_i = max(max_i, i)
        except Exception:
            dag_run.get_task_instance(storage_task_name).xcom_push(key='progress', value=max_i)
            raise


for key, group_by, downstreams in [
    ('mixpanel', 'regexp_replace(kite_metrics.kite_status_normalized.userId, \'\p{Cntrl}\')', [(False, load_athena_to_elastic), (True, load_athena_to_mixpanel)]),
    ('cio', 'regexp_replace(coalesce(kite_metrics.kite_status_normalized.properties__forgetful_metrics_id, kite_metrics.kite_status_normalized.userId), \'\p{Cntrl}\')', [(True, load_athena_to_cio)])
]:
    operator = insert_kite_status_normalized >> AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='insert_kite_status_1d_{}'.format(key),
        query='athena/queries/kite_status_1d.tmpl.sql',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        params={
            'key': key,
            'group_by': group_by,
            'languages': kite_status_config['languages'],
            'editors': kite_status_config['editors'],
            'lexical_providers': kite_status_config['lexical_providers'],
            'python_providers': kite_status_config['python_providers']
        },
        dag=dag,
    ) >> AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='generate_{}_json'.format(key),
        query='athena/queries/kite_status_1d_json.tmpl.sql',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        params={'key': key, 'languages': kite_status_config['languages']},
        dag=dag,
    )
    operator >> AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='cleanup_{}_table_json'.format(key),
        query='DROP TABLE kite_status_1d_{{params.key}}_{{ds_nodash}}_json',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        params={'key': key},
        dag=dag,
    )
    operator >> AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='cleanup_{}_table'.format(key),
        query='DROP TABLE kite_status_1d_{{params.key}}_{{ds_nodash}}',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        params={'key': key},
        dag=dag,
    )
    operator = operator >> S3ListOperator(
        aws_conn_id='aws_us_east_1',
        task_id='list_{}_json_files'.format(key),
        bucket='kite-metrics',
        prefix='athena/kite_status_1d_{{params.key}}/json/{{ds}}/',
        delimiter='/',
        params={'key': key},
        dag=dag,
    )

    def skip_older(execution_date, **ctx):
        return execution_date >= MP_START_DATE or (datetime.datetime(2020, 5, 19) < execution_date < datetime.datetime(2020, 5, 26))

    skip_older_operator = ShortCircuitOperator(
        task_id='skip_older_{}'.format(key),
        python_callable=skip_older,
        dag=dag,
        provide_context=True
    )

    for skip_older, downstream in downstreams:
        progress_operator = PythonOperator(
            python_callable=lambda ti, **kwargs: ti.xcom_push(key='progress', value=0),
            task_id='progress_storage_{}'.format(downstream.__name__),
            dag=dag,
            provide_context=True,
        )
        ds_operator = PythonOperator(
            python_callable=downstream,
            task_id=downstream.__name__,
            dag=dag,
            retries=4,
            provide_context=True,
            op_kwargs={'storage_task_name': 'progress_storage_{}'.format(downstream.__name__)}
        )
        if skip_older:
            operator >> skip_older_operator >> progress_operator >> ds_operator
        else:
            operator >> progress_operator >> ds_operator

insert_kite_status_normalized >> AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='update_activations_table',
    query='athena/queries/insert_activations.tmpl.sql',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    depends_on_past=True,
    dag=dag,
)

update_schema_dag = DAG(
    'update_kite_status_schema',
    default_args=default_args,
    description='Update the kite_status and kite_status_normalized schemas.',
    schedule_interval=None,
)

for table_name in ['kite_status', 'kite_status_segment', 'kite_status_normalized']:
    AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='drop_{}'.format(table_name),
        query='DROP TABLE {{params.table_name}}',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        dag=update_schema_dag,
        params={'table_name': table_name},
    ) >> AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='create_{}'.format(table_name),
        query='athena/tables/{}.tmpl.sql'.format(table_name),
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        dag=update_schema_dag,
        params={'schema': kite_status_schema, 'table_name': table_name}
    )
