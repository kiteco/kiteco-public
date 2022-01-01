import datetime
import io
import gzip
import json
import time

from airflow import DAG
from airflow.contrib.operators.aws_athena_operator import AWSAthenaOperator
from airflow.hooks.S3_hook import S3Hook
from airflow.operators.python_operator import PythonOperator
from airflow.models import Variable
import pytz
import requests
import yaml
from jinja2 import PackageLoader
import pkg_resources
from kite_airflow.slack_alerts import task_fail_slack_alert


default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime.datetime(2020, 1, 1),
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 1,
    'retry_delay': datetime.timedelta(minutes=5),
    'on_failure_callback': task_fail_slack_alert,
}

dag = DAG(
    'mixpanel_ingest',
    default_args=default_args,
    description='Mixpanel data ingest DAG.',
    schedule_interval='10 4 * * *',
    max_active_runs=1,
    jinja_environment_kwargs={
        'loader': PackageLoader('kite_airflow', 'templates')
    },
)


pacific = pytz.timezone('America/Los_Angeles')
people_schema = yaml.load(pkg_resources.resource_stream('kite_airflow', 'files/mixpanel_people.schema.yaml'), Loader=yaml.FullLoader)


def copy_profile_deltas(task_instance, execution_date, prev_execution_date_success, next_execution_date, **context):

    ex_day = execution_date.replace(hour=0, minute=0, second=0, microsecond=0)
    if prev_execution_date_success:
        ex_day = prev_execution_date_success.replace(hour=0, minute=0, second=0, microsecond=0) + datetime.timedelta(days=1)

    next_ex_day = next_execution_date.replace(hour=0, minute=0, second=0, microsecond=0)

    chunks = [ex_day]
    while chunks[-1] < next_ex_day:
        chunks.append(chunks[-1] + datetime.timedelta(hours=4))

    gz_file = io.BytesIO()

    with gzip.GzipFile(fileobj=gz_file, mode="w") as f:
        start_date = chunks.pop(0)
        for chunk in chunks:
            filters = []
            for cmp, dt in [['>=', start_date], ['<', chunk]]:
                filters.append('user.time {} {}'.format(cmp, 1000 * int(time.mktime(dt.astimezone(pacific).timetuple()))))
            start_date = chunk
            print(filters)
            script = 'function main() {{ return People().filter(function(user) {{ return {}; }})}}'.format(' && '.join(filters))
            res = requests.post('https://mixpanel.com/api/2.0/jql',
                                auth=(Variable.get('mixpanel_credentials', deserialize_json=True)['secret'], ''),
                                data={'script': script})
            if res.status_code != 200:
                raise Exception(res.text)

            for line in res.json():
                to_scrub = [line]
                while to_scrub:
                    curr = to_scrub.pop(0)
                    for key, value in list(curr.items()):
                        if isinstance(value, (dict, list)) and len(value) == 0:
                            del curr[key]
                        if isinstance(value, dict):
                            to_scrub.append(value)
                        if key.startswith('$'):
                            curr[key[1:]] = value
                            del curr[key]

                for ts_field in ['last_seen', 'time']:
                    pacific_ts = datetime.datetime.fromtimestamp(line[ts_field] / 1000).replace(tzinfo=pacific)
                    line[ts_field] = int(time.mktime(pacific_ts.astimezone(pytz.utc).timetuple()))

                f.write(json.dumps(line).encode('utf8'))
                f.write(b'\n')

    s3 = S3Hook('aws_us_east_1')
    key = 'mixpanel/people/raw/year={}/month={}/day={}/deltas.json.gz'.format(
      execution_date.year, execution_date.month, execution_date.day
    )
    s3.load_bytes(gz_file.getvalue(), key, 'kite-metrics')


PythonOperator(
    python_callable=copy_profile_deltas,
    task_id=copy_profile_deltas.__name__,
    dag=dag,
    retries=2,
    provide_context=True,
) >> AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='rollup_people',
    query='athena/queries/mixpanel_people_rollup.tmpl.sql',
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
    params={'schema': people_schema},
) >> AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='cleanup_rollup_table',
    query="DROP TABLE mixpanel_people_rollup_{{ds_nodash}}",
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
    params={'schema': people_schema},
) >> AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='update_people_table_location',
    query="""ALTER TABLE mixpanel_people
SET LOCATION 's3://kite-metrics/mixpanel/people/rollups/year={{execution_date.year}}/month={{execution_date.month}}/day={{execution_date.day}}/'""",
    output_location='s3://kite-metrics-test/athena-results/ddl',
    database='kite_metrics',
    dag=dag,
    params={'schema': people_schema},
)


ddl_dag = DAG(
    'mixpanel_ingest_schema_update',
    default_args=default_args,
    description='Mixpanel data schema definition.',
    schedule_interval=None,
    max_active_runs=1,
)

for table_name, s3_prefix in {'mixpanel_people_raw': 'mixpanel/people/raw', 'mixpanel_people': 'mixpanel/people/rollups'}.items():
    AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='drop_{}'.format(table_name),
        query='DROP TABLE {{params.table_name}}',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        dag=ddl_dag,
        params={'table_name': table_name},
    ) >> AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='create_{}'.format(table_name),
        query='athena/tables/mixpanel_people.tmpl.sql',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        dag=ddl_dag,
        params={
            'schema': people_schema,
            'table_name': table_name,
            's3_prefix': s3_prefix,
            'partitioned': table_name == 'mixpanel_people_raw',
            'json': table_name == 'mixpanel_people_raw',
        }
    )
