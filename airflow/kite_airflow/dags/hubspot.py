import logging
import datetime
import tempfile
import requests
import yaml
import json
import gzip
import re
import time

from airflow import DAG
from jinja2 import Template
import customerio

from airflow.contrib.operators.aws_athena_operator import AWSAthenaOperator
from airflow.hooks.S3_hook import S3Hook
from airflow.hooks.postgres_hook import PostgresHook
from airflow.operators.python_operator import PythonOperator
from airflow.models import Variable
from airflow.sensors.external_task_sensor import ExternalTaskSensor
from airflow.contrib.operators.s3_list_operator import S3ListOperator
from kite_airflow.dags.kite_status_1d import dag as kits_status_1d_dag, read_s3_json_files
from jinja2 import PackageLoader
import concurrent.futures
import pkg_resources
import kite_metrics
from kite_airflow.slack_alerts import task_fail_slack_alert


logger = logging.getLogger(__name__)

default_args = {
    'owner': 'airflow',
    'depends_on_past': True,
    'start_date': datetime.datetime(2020, 6, 28),
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 1,
    'retry_delay': datetime.timedelta(minutes=5),
    'on_failure_callback': task_fail_slack_alert,
}


DATA_LOC = 's3://kite-metrics/firehose/kite_status/'
PROD_RESULT_LOC_PREFIX = 's3://kite-metrics/athena-results'

kite_status_config = kite_metrics.load_context('kite_status')
LANGS = kite_status_config['languages']
EDITORS = kite_status_config['editors']

contat_props_tmpl = Template(pkg_resources.resource_string('kite_airflow', 'files/hubspot_contactprops.yaml').decode('utf8'))
contact_props_yaml = contat_props_tmpl.render(editors=EDITORS, langs=LANGS)
contact_props = yaml.load(contact_props_yaml, Loader=yaml.FullLoader)

dag = DAG(
    'hubspot_user_metrics',
    default_args=default_args,
    description='A simple tutorial DAG',
    schedule_interval='30 0 * * *',
    max_active_runs=1,
    jinja_environment_kwargs={
        'loader': PackageLoader('kite_airflow', 'templates')
    },
)

previous_dag_run_sensor = ExternalTaskSensor(
    task_id='previous_dag_run_sensor',
    dag=dag,
    external_dag_id=dag.dag_id,
    execution_delta=datetime.timedelta(days=1),
    mode='reschedule',
)

kite_status_dag_run_sensor = ExternalTaskSensor(
    task_id='kite_status_dag_run_sensor',
    dag=dag,
    execution_delta=datetime.timedelta(minutes=20),
    external_dag_id=kits_status_1d_dag.dag_id,
    mode='reschedule',
)

drop_intermediate_table = AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='drop_intermediate_table',
    query='DROP TABLE kite_metrics.hubspot_intermediate',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
)

create_intermediate_table = AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='create_intermediate_table',
    query='athena/tables/hubspot_intermediate.tmpl.sql',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
    params={'props': contact_props},
)

(previous_dag_run_sensor, kite_status_dag_run_sensor) >> drop_intermediate_table >> create_intermediate_table

insert_deltas = AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='insert_deltas',
    query='athena/queries/hubspot_delta.tmpl.sql',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    params={'props': contact_props},
    dag=dag,
)

insert_deltas >> AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='cleanup_delta_table',
    query='DROP TABLE hubspot_delta_{{ds_nodash}}',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
)
(previous_dag_run_sensor, kite_status_dag_run_sensor) >> insert_deltas

EMAIL_RE = re.compile(r'^\s*[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\s*$', re.I)


def write_contact_prop_data(ti, **context):
    props = [p['name'] for p in contact_props if 'label' in p]
    props.append('user_id')

    s3 = S3Hook('aws_us_east_1')
    buffer = []

    # Hubspot validates emails against some list of domain extensions. Go fetch a list to replicate that.
    domains_resp = requests.get('https://data.iana.org/TLD/tlds-alpha-by-domain.txt')
    domains = set([d.lower() for d in domains_resp.text.split('\n') if re.match('^[a-z]+$', d.lower())])
    counter = 0

    for file in sorted(ti.xcom_pull(task_ids='list_hubspot_json_files')):
        obj = s3.get_key(file, 'kite-metrics')
        for line in gzip.open(obj.get()['Body']):
            counter += 1
            if counter % 1000 == 0:
                logger.info('Processed {} records'.format(counter))

            rec = json.loads(line)
            email = rec['email']
            if not EMAIL_RE.match(email) or email.rsplit('.', 1)[1] not in domains:
                logger.info('Skipping invalid email address {}'.format(email))
                continue

            if any([rec.get('{}_percentage'.format(key)) is not None for key in LANGS]):
                rec['user_data_primary_language'] = max(LANGS, key=lambda x: rec.get('{}_percentage'.format(x)) or 0)

            if any([rec.get('python_edits_in_{}'.format(key)) for key in EDITORS]):
                rec['user_data_primary_python_editor'] = max(EDITORS, key=lambda x: rec.get('python_edits_in_{}'.format(x)) or 0)

            hs_props = {prop: rec[prop] for prop in props if rec.get(prop) is not None}
            hs_props['kite_lifecycle_stages'] = 'User'  # This property is called 'Source' in HS
            buffer.append({'email': email, 'properties': [{'property': prop, 'value': value} for prop, value in hs_props.items()]})

            if len(buffer) >= 100:
                make_hubspot_request('contacts/v1/contact/batch', buffer)
                buffer = []

    if buffer:
        make_hubspot_request('contacts/v1/contact/batch', buffer)


def copy_kite_users():
    pg_hook = PostgresHook(postgres_conn_id='community')
    s3 = S3Hook('aws_us_east_1')
    tf = tempfile.NamedTemporaryFile()
    pg_hook.copy_expert("COPY public.user (id, name, email) TO STDOUT WITH (FORMAT csv)", tf.name)
    s3.load_file(tf.name, 'enrichment/kite/users/users.csv', bucket_name='kite-metrics', replace=True)


copy_kite_users_operator = PythonOperator(
    python_callable=copy_kite_users,
    task_id=copy_kite_users.__name__,
    dag=dag,
)

setup_partitions = AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='setup_final_partitions',
    query='MSCK REPAIR TABLE hubspot_intermediate',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
)

[create_intermediate_table, insert_deltas] >> AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='insert_rollups',
    query='athena/queries/hubspot_rollup.tmpl.sql',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
    params={
        'scalar_props': [p for p in contact_props if 'agg' in p['sql']],
        'map_props': [p for p in contact_props if 'map_agg' in p['sql']],
        'scalar_time_rollups': set([prop['sql']['agg_days'] for prop in contact_props if 'agg_days' in prop['sql']]),
    },
) >> AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='cleanup_rollup_table',
    query='DROP TABLE hubspot_rollup_{{ds_nodash}}',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
) >> setup_partitions


(copy_kite_users_operator, setup_partitions) >> AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='create_hubspot_final_table',
    query='athena/queries/hubspot_final.tmpl.sql',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
    params={
        'scalar_props': [p for p in contact_props if 'map_agg' not in p['sql']],
        'map_props': [p for p in contact_props if 'map_agg' in p['sql']],
    },
) >> S3ListOperator(
    aws_conn_id='aws_us_east_1',
    task_id='list_hubspot_json_files',
    bucket='kite-metrics',
    prefix='athena/hubspot/final/{{ds}}/',
    delimiter='/',
    dag=dag,
) >> PythonOperator(
    python_callable=write_contact_prop_data,
    task_id=write_contact_prop_data.__name__,
    dag=dag,
    provide_context=True,
) >> AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='cleanup_final_table',
    query='DROP TABLE hubspot_final_{{ds_nodash}}',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
)


def write_cio_profile_attrs(task_instance, execution_date, dag_run, **context):
    cio_creds = Variable.get('cio_credentials', deserialize_json=True)
    iter_records = read_s3_json_files('kite-metrics', task_instance.xcom_pull(task_ids='list_cio_json_files'))

    def iter():
        for i, rec in enumerate(iter_records):
            if not rec['id'] or not all(ord(c) < 128 for c in rec['id']):
                continue
            if 'time_zone' in rec:
                rec['timezone'] = rec.pop('time_zone')
            yield i, rec

    def call_cio(item):
        i, kwargs = item
        customerio.CustomerIO(cio_creds['site_id'], cio_creds['api_key']).identify(**kwargs)
        return i

    queue_size = 100
    pool_size = 20
    futures = []
    records_iter = iter()
    max_i = 0
    has_values = True
    with concurrent.futures.ThreadPoolExecutor(max_workers=pool_size) as executor:
        while has_values:
            while len(futures) < queue_size:
                try:
                    futures.append(executor.submit(call_cio, next(records_iter)))
                except StopIteration:
                    has_values = False
                    break

            mode = concurrent.futures.FIRST_COMPLETED if has_values else concurrent.futures.ALL_COMPLETED
            done, not_done = concurrent.futures.wait(futures, timeout=6000, return_when=mode)
            futures = list(not_done)
            for future in done:
                i = future.result()
                if max_i > 0 and (i // 1000) > (max_i // 1000):
                    logger.info("Processed line {}".format(i))
                max_i = max(max_i, i)


setup_partitions >> AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='create_cio_table',
    query='athena/queries/cio_profile_attrs.tmpl.sql',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
    params={
        'props': ["country_name", "city_name", "subdivision_1_name", "time_zone"]
    },
) >> S3ListOperator(
    aws_conn_id='aws_us_east_1',
    task_id='list_cio_json_files',
    bucket='kite-metrics',
    prefix='athena/cio_profile_attrs/{{ds}}/',
    delimiter='/',
    dag=dag,
) >> PythonOperator(
    python_callable=write_cio_profile_attrs,
    task_id=write_cio_profile_attrs.__name__,
    dag=dag,
    provide_context=True,
) >> AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='cleanup_cio_table',
    query='DROP TABLE cio_profile_attrs_{{ds_nodash}}',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
)


USER_DATA_PROPGROUP_NAME = 'user_data'
MAX_TRIES = 3


def make_hubspot_request(path, data=None, method=None, tries=0):
    url = 'https://api.hubapi.com/{}?hapikey={}'.format(path, Variable.get('hubspot_apikey'))
    req_fn = getattr(requests, method) if method else (requests.post if data else requests.get)
    resp = req_fn(url, **({'json': data} if data else {}))
    tries = tries + 1
    if resp.status_code == 502 and tries < MAX_TRIES:
        logger.warn('Got 502 from Hubspot API, sleeping 60 seconds before retry.')
        time.sleep(60)
        return make_hubspot_request(path, data, method, tries)
    if resp.status_code >= 300:
        raise Exception('Error make hubspot request, code={}, response={}'.format(resp.status_code, resp.text))
    return resp


def update_contact_props():
    props = make_hubspot_request('properties/v1/contacts/properties').json()
    props_dict = {prop['name']: prop for prop in props if prop['groupName'] == USER_DATA_PROPGROUP_NAME}

    for prop in contact_props:
        if 'label' not in prop:
            continue
        prop = prop.copy()
        prop.pop('sql', None)
        prop['groupName'] = USER_DATA_PROPGROUP_NAME
        if prop['name'] not in props_dict:
            make_hubspot_request('properties/v1/contacts/properties', prop)
            continue
        if {k: v for k, v in props_dict[prop['name']].items() if k in prop} == prop:
            continue
        make_hubspot_request('properties/v1/contacts/properties/named/{}'.format(prop['name']), prop, 'put')


update_contact_props_operator = PythonOperator(
    python_callable=update_contact_props,
    task_id=update_contact_props.__name__,
    dag=dag,
)

previous_dag_run_sensor >> update_contact_props_operator
