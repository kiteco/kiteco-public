from airflow import DAG
import datetime
from airflow.hooks.S3_hook import S3Hook
from airflow.contrib.hooks.aws_sqs_hook import SQSHook
import json
import gzip
import hashlib
import collections
import codecs
import logging
import uuid
import csv
import time
from airflow.operators.python_operator import PythonOperator
import requests
from airflow.contrib.operators.aws_athena_operator import AWSAthenaOperator
from jinja2 import PackageLoader
from kite_airflow.slack_alerts import task_fail_slack_alert


logger = logging.getLogger(__name__)

BUCKET = 'kite-youtube-data'
SCRATCH_SPACE_LOC = 's3://{}/athena-scratch-space/'.format(BUCKET)


def iter_s3_file(s3_hook, bucket, key):
    json_file = s3_hook.get_key(key, BUCKET)
    for line in gzip.open(json_file.get()['Body']):
        yield json.loads(line)


youtube_search_dag = DAG(
    'youtube_search',
    description='Find new Youtube channels.',
    default_args={
        'retries': 1,
        'retry_delay': datetime.timedelta(minutes=5),
        'start_date': datetime.datetime(2020, 11, 6),
        'on_failure_callback': task_fail_slack_alert,
    },
    schedule_interval='0 4 * * *',
    max_active_runs=1,
    jinja_environment_kwargs={
        'loader': PackageLoader('kite_airflow', 'templates')
    },
)

schema_operators = []
for table in ['youtube_queries', 'youtube_searches', 'youtube_channels', 'youtube_channel_details', 'youtube_socialblade_stats']:
    drop_op = AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='drop_table_{}'.format(table),
        query='DROP TABLE IF EXISTS {}'.format(table),
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_youtube_crawl',
        dag=youtube_search_dag
    )

    create_op = AWSAthenaOperator(
        aws_conn_id='aws_us_east_1',
        task_id='create_table_{}'.format(table),
        query='athena/tables/{}.tmpl.sql'.format(table),
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database='kite_youtube_crawl',
        dag=youtube_search_dag,
    )

    drop_op >> create_op
    schema_operators.append(create_op)

BATCH_SIZE = 100
MAX_RELATED_GENERATION = 1

get_queries_op = AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='get_queries',
    query='SELECT q.*, s.query IS NOT NULL AS searched FROM youtube_queries q LEFT OUTER JOIN youtube_searches s ON (q.query=s.query) ORDER BY q.count DESC',
    output_location=SCRATCH_SPACE_LOC,
    database='kite_youtube_crawl',
    dag=youtube_search_dag,
)
schema_operators >> get_queries_op

get_existing_channels_op = AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='get_existing_channels',
    query='SELECT id FROM youtube_channels',
    output_location=SCRATCH_SPACE_LOC,
    database='kite_youtube_crawl',
    dag=youtube_search_dag,
)
schema_operators >> get_existing_channels_op


def get_scratch_space_csv(s3hook, ti, task_id):
    filename = ti.xcom_pull(task_ids=task_id)
    s3key = s3hook.get_key('athena-scratch-space/{}.csv'.format(filename), BUCKET)
    return csv.DictReader(codecs.getreader("utf-8")(s3key.get()['Body']))


def write_gzip_string_to_s3(s3hook, contents, key, bucket):
    s3hook.load_string(gzip.compress(contents.encode('utf8')), key, bucket)


def youtube_crawl(ti, ts_nodash, **kwargs):
    s3 = S3Hook('aws_us_east_1')
    ex_channels = {c['id'] for c in get_scratch_space_csv(s3, ti, get_existing_channels_op.task_id)}

    curr_time = datetime.datetime.now()
    queries = get_scratch_space_csv(s3, ti, get_queries_op.task_id)
    selected_queries = [q for q in queries if q['searched'] == 'false'][:BATCH_SIZE]
    all_queries = {q['query'] for q in queries}
    search_records = []
    new_channels = []
    new_queries = []

    try:
        for query in selected_queries:
            print("Running query {}".format(query['query']))
            query_hash = hashlib.md5(query['query'].encode('utf8')).hexdigest()

            # resp = requests.get('https://serpapi.com/search.json',
            #     params={'engine': 'youtube', 'search_query': query['query'], 'api_key': 'XXXXXXX'})
            resp = requests.get("https://www.googleapis.com/youtube/v3/search", params={
                "key": "XXXXXXX",
                "q": query['query'],
                "part": "snippet",
                "maxResults": "50"
            }, headers={'content-type': 'application/json'})

            if resp.status_code != 200:
                print("Error from SerpAPI: {} {}".format(resp.status_code, resp.text))
                raise Exception()

            resp_json = resp.json()

            # if 'video_results' not in resp.json():
            if 'items' not in resp_json:
                print("No results for {}".format(query['query']))
                continue

            response_key = 'search_responses/{}/{}.json.gz'.format(query_hash, ts_nodash)
            s3.load_bytes(gzip.compress(resp.text.encode('utf8')), response_key, BUCKET, replace=True)

            # all_channels = {v['channel']['link'] for v in resp_json['video_results'] if 'link' in v['channel']}
            all_channels = {'https://www.youtube.com/channel/{}'.format(v['snippet']['channelId']) for v in resp_json['items']}
            n_new_channels = len(all_channels - ex_channels)

            for c in all_channels - ex_channels:
                new_channels.append({
                    'id': c,
                    'query': query['query'],
                    'timestamp': curr_time.isoformat(),
                })
                ex_channels.add(c)

            for key in resp_json:
                if not key.startswith('searches_related_to_'):
                    continue
                for search in resp_json[key]['searches']:
                    if search['query'] not in all_queries:
                        new_queries.append({
                            'query': search['query'],
                            'seed': False,
                            'generation': int(query.get('generation') or 0) + 1,
                            'parent': query['query']
                        })

            search_records.append({
                'query': query['query'],
                'query_hash': query_hash,
                'timestamp': curr_time.isoformat(),
                'total': len(all_channels),
                'unique': n_new_channels,
            })

    finally:
        for key, objs in [('channels', new_channels), ('search_queries', new_queries), ('searches', search_records)]:
            if objs:
                contents = gzip.compress('\n'.join([json.dumps(obj) for obj in objs]).encode('utf8'))
                s3.load_bytes(contents, '{}/{}-{}.json.gz'.format(key, ts_nodash, uuid.uuid4().hex), BUCKET)


youtube_crawl_op = PythonOperator(
    python_callable=youtube_crawl,
    task_id=youtube_crawl.__name__,
    dag=youtube_search_dag,
    provide_context=True,
)
(get_queries_op, get_existing_channels_op) >> youtube_crawl_op

get_new_channels_op = AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='get_new_channels',
    query='''SELECT DISTINCT c.id
             FROM youtube_channels c
             LEFT OUTER JOIN youtube_channel_details d ON (
              concat('https://www.youtube.com/channel/', d.id)=c.id
              OR concat('https://www.youtube.com/user/', d.forUsername)=c.id
            )
            WHERE d.id IS NULL AND d.forUsername IS NULL''',
    output_location=SCRATCH_SPACE_LOC,
    database='kite_youtube_crawl',
    dag=youtube_search_dag,
)


def chunks(lst, n):
    """Yield successive n-sized chunks from lst."""
    for i in range(0, len(lst), n):
        yield lst[i:i + n]


def get_channel_details(ti, ts_nodash, **kwargs):
    s3 = S3Hook('aws_us_east_1')
    new_channels = {c['id'] for c in get_scratch_space_csv(s3, ti, get_new_channels_op.task_id)}

    channels_by_type = collections.defaultdict(list)
    for channel in new_channels:
        c_parts = channel.split('/')
        channels_by_type[c_parts[-2]].append(c_parts[-1])

    print("Getting channel details for {} new channels and {} new users".format(
        len(channels_by_type['channel']),
        len(channels_by_type['user']))
    )

    c_details = []

    url = "https://www.googleapis.com/youtube/v3/channels"
    generic_params = {
        "part": ["statistics", "snippet", "contentDetails", "status"],
        "key": "XXXXXXX",
    }

    try:
        for username in channels_by_type.get('user', []):
            params = {'forUsername': username}
            params.update(generic_params)

            resp = requests.get(url, params=params, headers={'content-type': 'application/json'})
            if not resp.json().get('items'):
                print("Failed to get user: {}".format(username))
                c_details.append({'forUsername': username})
                continue

            for item in resp.json()['items']:
                item['forUsername'] = username
                c_details.append(item)

        for chunk in chunks(channels_by_type.get('channel', []), 50):
            params = {'id': ','.join(chunk)}
            params.update(generic_params)

            resp = requests.get(url, params=params, headers={'content-type': 'application/json'})
            if not resp.json().get('items'):
                print("Failed to get channels: {}".format(', '.join(chunk)))
                continue
            for item in resp.json()['items']:
                c_details.append(item)
    finally:
        print("Loading channel details for {} channels".format(len(c_details)))
        contents = gzip.compress('\n'.join([json.dumps(obj) for obj in c_details]).encode('utf8'))
        s3.load_bytes(contents, 'channel_details/{}-{}.json.gz'.format(ts_nodash, uuid.uuid4().hex), BUCKET)


get_channel_details_op = PythonOperator(
    python_callable=get_channel_details,
    task_id=get_channel_details.__name__,
    dag=youtube_search_dag,
    provide_context=True,
)
youtube_crawl_op >> get_new_channels_op >> get_channel_details_op

get_new_socialblade_channels = AWSAthenaOperator(
    aws_conn_id='aws_us_east_1',
    task_id='get_new_socialblade_channels',
    query='''SELECT DISTINCT c.id
             FROM youtube_channel_details c
             LEFT OUTER JOIN youtube_socialblade_stats sb ON c.id=sb.id
             WHERE sb.id IS NULL AND CAST(c.statistics.viewCount AS bigint) > 100000''',
    output_location=SCRATCH_SPACE_LOC,
    database='kite_youtube_crawl',
    dag=youtube_search_dag,
)

QUEUE_URL = 'https://sqs.us-east-1.amazonaws.com/XXXXXXX/queue-youtube-socialblade'


def enqueue_socialblade_channels(ti, ts_nodash, **kwargs):
    s3 = S3Hook('aws_us_east_1')
    sqs_hook = SQSHook('aws_us_east_1')

    new_channels = {c['id'] for c in get_scratch_space_csv(s3, ti, get_new_socialblade_channels.task_id)}
    print('Enqueuing {} channels'.format(len(new_channels)))
    sqs_hook.get_conn().purge_queue(QueueUrl=QUEUE_URL)

    # Sleep to allow purge to complete
    time.sleep(60)
    for channel in new_channels:
        sqs_hook.send_message(QUEUE_URL, channel)


enqueue_socialblade_channels_op = PythonOperator(
    python_callable=enqueue_socialblade_channels,
    task_id=enqueue_socialblade_channels.__name__,
    dag=youtube_search_dag,
    provide_context=True,
)

get_channel_details_op >> get_new_socialblade_channels >> enqueue_socialblade_channels_op
