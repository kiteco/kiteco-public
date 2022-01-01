import datetime
import logging
import time

from airflow import DAG
from airflow.operators.python_operator import PythonOperator
from airflow.models import Variable
from jinja2 import PackageLoader
import mixpanel

from kite_airflow.dags.hubspot import make_hubspot_request
from kite_airflow.plugins.google import GoogleSheetsRangeOperator
from kite_airflow.slack_alerts import task_fail_slack_alert


logger = logging.getLogger(__name__)


default_args = {
    'owner': 'airflow',
    'depends_on_past': True,
    'start_date': datetime.datetime(2021, 1, 7),
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 1,
    'retry_delay': datetime.timedelta(minutes=5),
    'on_failure_callback': task_fail_slack_alert,
}


dag = DAG(
    'hubspot_companies',
    default_args=default_args,
    description='Sychronizes user company data from hubspot to other systems.',
    schedule_interval='0 12 * * *',
    max_active_runs=1,
    jinja_environment_kwargs={
        'loader': PackageLoader('kite_airflow', 'templates')
    },
)


MP_COMPANY_PROP = 'Company name'


def write_company_assignments(ti, **ctx):
    mp_consumer = mixpanel.BufferedConsumer(max_size=100)
    mp_client = mixpanel.Mixpanel(Variable.get('mixpanel_credentials', deserialize_json=True)['token'], consumer=mp_consumer)

    logger.info("Fetching company list")
    supported_companies = [rec[0] for rec in ti.xcom_pull(task_ids='get_companies_sheet')['values']]
    for company in supported_companies:
        logger.info("Starting processing for company {}".format(company))
        params = {
            'limit': 100,
            'filterGroups': [{'filters': [
                {'propertyName': 'company', 'operator': 'EQ', 'value': company},
                {'propertyName': 'user_id', 'operator': 'HAS_PROPERTY'}
            ]}],
            'properties': ['user_id'],
        }
        n_done = 0
        while True:
            resp = make_hubspot_request('crm/v3/objects/contacts/search', params).json()
            if resp['total'] == 0:
                raise Exception('No results for company "{}". Is it mis-spelled?'.format(company))

            for res in resp['results']:
                mp_client.people_set(
                    res['properties']['user_id'],
                    {MP_COMPANY_PROP: company},
                    meta={'$ignore_time': 'true', '$ip': 0})
                n_done += 1

            logger.info("  {} / {} records processed".format(n_done, resp['total']))

            after = resp.get('paging', {}).get('next', {}).get('after')
            if not after:
                break
            params['after'] = after
            time.sleep(20)
    mp_consumer.flush()


GoogleSheetsRangeOperator(
    gcp_conn_id='google_cloud_kite_dev',
    spreadsheet_id='XXXXXXX',
    range="'Companies to Import to Mixpanel'!CompanyNames",
    task_id='get_companies_sheet',
    dag=dag,
) >> PythonOperator(
    python_callable=write_company_assignments,
    task_id=write_company_assignments.__name__,
    dag=dag,
    provide_context=True,
)
