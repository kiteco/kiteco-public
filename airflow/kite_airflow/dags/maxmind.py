from airflow import DAG
import ipaddress
import datetime
import requests
import io
import os
import csv
import zipfile
from airflow.models import Variable
from airflow.hooks.S3_hook import S3Hook
from airflow.operators.python_operator import PythonOperator
from airflow.contrib.operators.aws_athena_operator import AWSAthenaOperator
from kite_airflow.s3_utils import S3DeletePrefixOperator
from jinja2 import PackageLoader
from kite_airflow.slack_alerts import task_fail_slack_alert


default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime.datetime(2020, 6, 12),
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 0,
    'retry_delay': datetime.timedelta(minutes=5),
    'on_failure_callback': task_fail_slack_alert,
}

dag = DAG(
    'maxmind_geolite2',
    description='Load the Maxmind Geolite2 database.',
    default_args=default_args,
    schedule_interval='0 0 * * 0',
    jinja_environment_kwargs={
        'loader': PackageLoader('kite_airflow', 'templates')
    },
)

maxmind_url = 'https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-{}-CSV&license_key={}&suffix=zip'
maxmind_files = [
    'GeoLite2-Country-Blocks-IPv4',
    'GeoLite2-Country-Blocks-IPv6',
    'GeoLite2-Country-Locations-en',
]
bucket_name = 'kite-metrics'
key_prefix_template = 'enrichment/maxmind/{prefix}/{dataset}/{ds}/{filename}/'
key_template = key_prefix_template + '{filename}.csv'


def maxmind_operator_fn(ds, **context):
    for dataset in ['city', 'country']:
        mm_resp = requests.get(maxmind_url.format(dataset.title(), Variable.get('maxmind_license_key')))
        mm_zipfile = zipfile.ZipFile(io.BytesIO(mm_resp.content))
        s3 = S3Hook('aws_us_east_1')

        for path in mm_zipfile.namelist():
            if not path.endswith('.csv'):
                continue
            mm_file = mm_zipfile.open(path)
            filename = os.path.splitext(os.path.basename(path))[0]
            s3.load_file_obj(mm_file, key_template.format(prefix='raw', dataset=dataset, ds=ds, filename=filename), bucket_name=bucket_name, replace=True)

        ipv4_path = [p for p in mm_zipfile.namelist() if p.endswith('GeoLite2-{}-Blocks-IPv4.csv'.format(dataset.title()))][0]
        ipv4_file = io.TextIOWrapper(mm_zipfile.open(ipv4_path, 'r'))
        ipv4_reader = csv.DictReader(ipv4_file)
        ipv4_output = io.StringIO()
        ipv4_writer = csv.DictWriter(ipv4_output, ipv4_reader.fieldnames + ['address', 'mask'])
        for rec in ipv4_reader:
            net = ipaddress.IPv4Network(rec['network'])
            rec['address'] = int(net.network_address)
            rec['mask'] = int(net.netmask)
            ipv4_writer.writerow(rec)
        key = key_template.format(prefix='expanded', dataset=dataset, ds=ds, filename='GeoLite2-{}-Blocks-IPv4'.format(dataset))
        s3.load_string(ipv4_output.getvalue(), key, bucket_name=bucket_name, replace=True)


maxmind_operator = PythonOperator(
    python_callable=maxmind_operator_fn,
    task_id='load_maxmind_to_s3',
    dag=dag,
    provide_context=True,
)

for dataset in ['city']:
    maxmind_operator >> AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='create_{}_names_table'.format(dataset),
        query='''CREATE EXTERNAL TABLE kite_metrics.maxmind_{{params.dataset}}_names_{{ds_nodash}} (
            geoname_id string,
            locale_code string,
            continent_code string,
            continent_name string,
            country_iso_code string,
            country_name string,
            subdivision_1_iso_code string,
            subdivision_1_name string,
            subdivision_2_iso_code string,
            subdivision_2_name string,
            city_name string,
            metro_code string,
            time_zone string,
            is_in_european_union string)
        ROW FORMAT SERDE 'org.apache.hadoop.hive.serde2.OpenCSVSerde'
        LOCATION 's3://{{params.bucket}}/{{params.key_prefix_template.format(ds=ds, dataset=params.dataset, prefix='raw', filename=params.filename)}}'
        TBLPROPERTIES ('skip.header.line.count'='1')
        ''',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        dag=dag,
        params={
            'bucket': bucket_name,
            'key_prefix_template': key_prefix_template,
            'filename': 'GeoLite2-{}-Locations-en'.format(dataset.title()),
            'dataset': dataset},
    ) >> AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='create_ipv4_{}_table'.format(dataset),
        query='''CREATE EXTERNAL TABLE kite_metrics.maxmind_ipv4_{{params.dataset}}_{{ds_nodash}} (
            network string,
            geoname_id string,
            registered_country_geoname_id string,
            represented_country_geoname_id string,
            is_anonymous_proxy string,
            is_satellite_provider string,
            postal_code string,
            latitude string,
            longitude string,
            accuracy_radius string,
            address bigint,
            mask bigint)
        ROW FORMAT SERDE 'org.apache.hadoop.hive.serde2.OpenCSVSerde'
        LOCATION 's3://{{params.bucket}}/{{params.key_prefix_template.format(ds=ds, dataset=params.dataset, prefix='expanded', filename=filename)}}'
        TBLPROPERTIES ('skip.header.line.count'='1')
        ''',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        dag=dag,
        params={
            'bucket': bucket_name,
            'key_prefix_template': key_prefix_template,
            'filename': 'GeoLite2-{}-Blocks-IPv4'.format(dataset.title()),
            'dataset': dataset
        },
    ) >> AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='drop_ipv4_{}_table'.format(dataset),
        query='''DROP TABLE kite_metrics.maxmind_{{params.dataset}}_ipv4''',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        dag=dag,
        params={
            'dataset': dataset
        },
    ) >> S3DeletePrefixOperator(
        aws_conn_id='aws_us_east_1',
        task_id='prepare_ipv4_{}_join_destination'.format(dataset),
        bucket='kite-metrics',
        keys='enrichment/maxmind/join/{{params.dataset}}/ipv4/',
        params={'dataset': dataset},
        dag=dag,
    ) >> AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='create_ipv4_{}_join_table'.format(dataset),
        query='''CREATE TABLE kite_metrics.maxmind_{{params.dataset}}_ipv4
        WITH (format='PARQUET',
              parquet_compression='SNAPPY',
              external_location = 's3://{{params.bucket}}/enrichment/maxmind/join/{{params.dataset}}/ipv4/')
        AS SELECT
            kite_metrics.maxmind_city_names_{{ds_nodash}}.country_iso_code country_iso_code,
            kite_metrics.maxmind_city_names_{{ds_nodash}}.country_name country_name,
            kite_metrics.maxmind_city_names_{{ds_nodash}}.subdivision_1_name subdivision_1_name,
            kite_metrics.maxmind_city_names_{{ds_nodash}}.city_name city_name,
            kite_metrics.maxmind_city_names_{{ds_nodash}}.time_zone time_zone,
            kite_metrics.maxmind_ipv4_city_{{ds_nodash}}.address address,
            kite_metrics.maxmind_ipv4_city_{{ds_nodash}}.mask mask
        FROM kite_metrics.maxmind_ipv4_city_{{ds_nodash}}
        JOIN kite_metrics.maxmind_city_names_{{ds_nodash}}
            ON kite_metrics.maxmind_ipv4_city_{{ds_nodash}}.geoname_id = kite_metrics.maxmind_city_names_{{ds_nodash}}.geoname_id
        ''',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        dag=dag,
        params={'bucket': bucket_name, 'dataset': dataset},
    ) >> AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='cleanup_ipv4_{}_table'.format(dataset),
        query='''DROP TABLE kite_metrics.maxmind_ipv4_{{params.dataset}}_{{ds_nodash}}''',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        dag=dag,
        params={
            'dataset': dataset
        },
    ) >> AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='cleanup_{}_names_table'.format(dataset),
        query='''DROP TABLE kite_metrics.maxmind_{{params.dataset}}_names_{{ds_nodash}}''',
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_metrics',
        dag=dag,
        params={
            'dataset': dataset
        },
    )
