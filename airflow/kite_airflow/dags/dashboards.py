import requests
import time
from airflow.models import Variable
from airflow import DAG
import pendulum
import datetime
import json
from airflow.operators.python_operator import PythonOperator
from kite_airflow.slack_alerts import task_fail_slack_alert


KIBANA_VERSION = '7.9.3'
KIBANA_URL = XXXXXXX
SLACK_URL = 'https://slack.com/api/files.upload'


local_tz = pendulum.timezone('America/Los_Angeles')


default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime.datetime(2020, 10, 27, tzinfo=local_tz),
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 0,
    'on_failure_callback': task_fail_slack_alert,
}

dag = DAG(
    'slack_dashboards',
    default_args=default_args,
    description='Render and post dashboards to Slack.',
    schedule_interval='0 10 * * *',
)


def dashboards(conf, **context):
    import logging
    logger = logging.getLogger("airflow.task")

    kibana_requests_kwargs = {'headers': {'kbn-version': KIBANA_VERSION}, 'auth': ('elastic', Variable.get('elastic_password'))}

    dashboards = Variable.get("slack_dashboards", deserialize_json=True)
    enqueued = []
    for dashboard in dashboards:
        res = requests.post(dashboard['url'], **kibana_requests_kwargs)
        if res.status_code != 200:
            raise Exception("Error requesting dashboard, config={}, code={}, response={}".format(json.dumps(dashboard), res.status_code, res.text))
        logger.info("ENQUEUE RES={}".format(res.json()))
        enqueued.append(res.json())

    errors = []
    for dashboard, rendered_url in zip(dashboards, enqueued):
        logger.info('Waiting for dashboard "{}"'.format(dashboard['slackParams']['title']))
        while True:
            res = requests.get("{}{}".format(KIBANA_URL, rendered_url['path']), **kibana_requests_kwargs)
            if res.status_code == 503:
                logger.info('Received 503 response, sleeping.')
                time.sleep(60)
                continue
            elif res.status_code != 200:
                errors.append('Error fetching rendered dashboard, config={}, code={}, response={}'.format(json.dumps(dashboard), res.status_code, res.text))
                break

            logger.info('Kibana response: code={}, response={}'.format(res.status_code, res.content))
            filename = dashboard['slackParams']['filename']
            logger.info('Slack request: files={}, headers={}, url={}'.format({
                    'file': (filename, res.content),
                    **{k: (None, v) for k, v in dashboard['slackParams'].items()},
                }, {'Authorization': 'Bearer {}'.format(Variable.get('slack_token'))}, SLACK_URL))
            slack_res = requests.post(
                SLACK_URL,
                files={
                    'file': (filename, res.content),
                    **{k: (None, v) for k, v in dashboard['slackParams'].items()},
                },
                headers={'Authorization': 'Bearer {}'.format(Variable.get('slack_token'))}
            )
            logger.info('Slack response: code={}, response={}'.format(slack_res.status_code, slack_res.text))
            break

    if errors:
        raise Exception('\n'.join(errors))


dashboards_operator = PythonOperator(
    python_callable=dashboards,
    task_id='dashboards',
    dag=dag,
    provide_context=True,
)
