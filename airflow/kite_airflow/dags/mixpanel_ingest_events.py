import datetime
import io
import gzip
import json
import time

from airflow import DAG
from airflow.hooks.S3_hook import S3Hook
from airflow.operators.python_operator import PythonOperator
from airflow.models import Variable
import pendulum
import requests
from jinja2 import PackageLoader
from kite_airflow.slack_alerts import task_fail_slack_alert


pacific = pendulum.timezone('America/Los_Angeles')

default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime.datetime(2020, 1, 1, tzinfo=pacific),
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 1,
    'retry_delay': datetime.timedelta(minutes=5),
    'on_failure_callback': task_fail_slack_alert,
}


dag = DAG(
    'mixpanel_ingest_events',
    default_args=default_args,
    description='Mixpanel events ingest DAG.',
    schedule_interval='30 * * * *',
    max_active_runs=6,
    jinja_environment_kwargs={
        'loader': PackageLoader('kite_airflow', 'templates')
    },
)


def copy_mp_raw_events(task_instance, execution_date, **context):
    pac_date = execution_date.astimezone(pacific)
    pac_hour = pac_date.replace(minute=0, second=0, microsecond=0)

    script = '''function main() {{
        return Events({{from_date: "{date}", to_date: "{date}"}}).filter(function(event) {{
            return !event.name.startsWith("kite_status") && event.time >= {start} && event.time < {end};
        }});
    }}'''.format(
        date=pac_date.strftime('%Y-%m-%d'),
        start=1000 * int(time.mktime(pac_hour.timetuple())),
        end=1000 * int(time.mktime((pac_hour + datetime.timedelta(hours=1)).timetuple())),
    )
    print(script)
    res = requests.post('https://mixpanel.com/api/2.0/jql',
                        auth=(Variable.get('mixpanel_credentials', deserialize_json=True)['secret'], ''),
                        data={'script': script},)

    if res.status_code != 200:
        raise Exception(res.text)

    files = {}

    for line in res.json():
        to_scrub = [line]
        while to_scrub:
            curr = to_scrub.pop(0)
            for key, value in list(curr.items()):
                if isinstance(value, (dict, list)) and len(value) == 0:
                    del curr[key]
                    continue
                if isinstance(value, dict):
                    to_scrub.append(value)
                    continue
                if key.startswith('$'):
                    curr[key[1:]] = value
                    del curr[key]

        pacific_ts = datetime.datetime.fromtimestamp(line['time'] / 1000).replace(tzinfo=pacific)
        utc_ts = pacific_ts.astimezone(pendulum.timezone('UTC'))
        line['time'] = int(time.mktime(utc_ts.timetuple()))

        file_key = 'year={}/month={}/day={}/hour={}/event={}'.format(utc_ts.year, utc_ts.month, utc_ts.day, utc_ts.hour, line['name'])
        if file_key not in files:
            b_io = io.BytesIO()
            files[file_key] = (b_io, gzip.GzipFile(fileobj=b_io, mode="w"))

        files[file_key][1].write(json.dumps(line).encode('utf8'))
        files[file_key][1].write(b'\n')

    s3 = S3Hook('aws_us_east_1')
    for prefix, (b_io, gz_file) in files.items():
        gz_file.close()
        s3.load_bytes(b_io.getvalue(), 'mixpanel/events/raw/{}/events.json.gz'.format(prefix), 'kite-metrics', replace=True)


PythonOperator(
    python_callable=copy_mp_raw_events,
    task_id=copy_mp_raw_events.__name__,
    dag=dag,
    retries=2,
    provide_context=True,
)
