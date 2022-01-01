import boto3
import datetime
import gzip
import os
from elasticsearch.helpers import bulk
from elasticsearch import Elasticsearch, RequestsHttpConnection
from requests_aws4auth import AWS4Auth
from iter_actions import iter_actions

INDEX_GRANULARITY = datetime.timedelta(days=10)

region = 'us-east-1'  # e.g. us-west-1
service = 'es'
credentials = boto3.Session().get_credentials()
awsauth = AWS4Auth(credentials.access_key,
                   credentials.secret_key,
                   region,
                   service,
                   session_token=credentials.token)
host = ('https://search-kite-telemetry-dev-3-XXXXXXX'
        '.us-east-1.es.amazonaws.com')


s3 = boto3.client('s3')
es_client = Elasticsearch(
    hosts=[host],
    http_auth=awsauth,
    use_ssl=True,
    verify_certs=True,
    connection_class=RequestsHttpConnection
)


# Lambda execution starts here
def handler(event, context):
    for record in event['Records']:
        # Get the bucket name and key for the new file
        bucket = record['s3']['bucket']['name']
        key = record['s3']['object']['key']

        date = parse_date_from_key(key)
        shard = get_index_shard(date, INDEX_GRANULARITY)

        # Get, read, and split the file into lines
        obj = s3.get_object(Bucket=bucket, Key=key)
        body = gzip.open(obj['Body'])
        bulk(es_client, iter_actions(shard, body, **record))


def get_index_shard(date, granularity, epoch=datetime.date(1970, 1, 1)):
    rounded = epoch + (date - epoch) // granularity * granularity
    return rounded.isoformat()


def parse_date_from_key(key):
    parts = key.split('/')
    if parts[0] == "firehose":
        # firehose/{stream_name}/{yyyy}/{mm}/{dd}/{hh}/{random_filename}.gz
        yyyy, mm, dd = parts[2:5]
        return datetime.date(year=int(yyyy), month=int(mm), day=int(dd))
    elif parts[0] == "segment-logs":
        # segment-logs/{source_id}/{unix_ts_ms}/{random_filename}.gz
        unix_ts_ms = int(parts[2])
        unix_ts = unix_ts_ms / 1000
        return datetime.datetime.utcfromtimestamp(unix_ts).date()
    raise ValueError("unhandled S3 prefix")
