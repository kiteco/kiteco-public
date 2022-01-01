import asyncio
import datetime
import re
import uuid

from telemetry_loader.extractors import extract_s3
from telemetry_loader.transformers.kite_status import transform_elastic_kite_status_1d
from telemetry_loader.transformers.kite_status import transform_mixpanel_kite_status_1d
from telemetry_loader.transformers.kite_status import transform_elastic_kite_status
from telemetry_loader.transformers.index_metrics import transform_elastic_index_metrics

from telemetry_loader.config import s3_var
from telemetry_loader.loaders.elastic import load_elastic
from telemetry_loader.loaders.mixpanel import load_mixpanel

from telemetry_loader.streams.core import bundle
from telemetry_loader.streams.core import null_consumer
from telemetry_loader.streams.core import fork
from telemetry_loader.streams.core import pipe
from telemetry_loader.streams.core import side_effect
from telemetry_loader.streams.pipes import csv_pipe
from telemetry_loader.streams.pipes import json_pipe
from telemetry_loader.streams.pipes import progress
from telemetry_loader.extractors.athena.db import Queries, run_query

from telemetry_loader.connections import connections_var, Connections

import logging
logger = logging.getLogger(__name__)


def kite_status_1d(**kwargs):
    elastic, mixpanel = extract_s3()(**kwargs) | progress(logger.info) | csv_pipe() | fork(2)
    elastic = elastic | transform_elastic_kite_status_1d | load_elastic
    mixpanel = mixpanel | transform_mixpanel_kite_status_1d | null_consumer  # load_mixpanel
    return bundle(elastic, mixpanel)


def kite_status(**kwargs):
    return extract_s3(compressed=True, range_size=8e6)(**kwargs) | progress(logger.info) | transform_elastic_kite_status | load_elastic


def index_metrics(**kwargs):
    return extract_s3(compressed=True)(**kwargs) | progress(logger.info) | transform_elastic_index_metrics | load_elastic


def aws_cost_reports(bucket, key):
    alias = 'aws_cost_reports_alias'
    date_range = re.search(r'\d{8}-\d{8}', key)
    date_range_prefix = 'aws_cost_reports-{}-'.format(date_range.group())
    index_name = 'aws_cost_reports-{}-{}'.format(date_range.group(), str(uuid.uuid4()))

    @pipe
    def cost_report_transformer(line):
        if line.get('product/normalizationSizeFactor') == 'NA':
            del line['product/normalizationSizeFactor']
        return {'_index': index_name, '_op_type': 'index', '_source': line}

    @side_effect
    def update_alias(_):
        client = connections_var.get().elasticsearch
        ex = client.indices.get_alias(name=alias)
        rm = [index for index in ex.keys() if index.startswith(date_range_prefix)]
        body = {'actions': [{'add': {'index': index_name, 'alias': alias}}, *[{'remove': {'index': idx, 'alias': alias}} for idx in rm]]}
        client.indices.update_aliases(body=body)
        if rm:
            client.indices.delete(index=','.join(rm))

    return extract_s3(compressed=True)(bucket, key) | progress(logger.info) | csv_pipe() | cost_report_transformer | \
        load_elastic | update_alias


def server_logs(**kwargs):
    @json_pipe()
    def server_logs_transformer(line):
        return {'_index': 'server_logs_write', '_source': line}

    return extract_s3(compressed=True)(**kwargs) | progress(logger.info) | server_logs_transformer | load_elastic


# Lambda execution starts here
def lambda_handler(event, context):
    if event['source'] == 'aws.events':
        return lambda_event_handler(event)

    for record in event['Records']:
        # Get the bucket name and key for the new file
        lambda_event_source_handlers[record['eventSource']](record)


def lambda_s3_handler(record):
    return s3_handler(record['s3']['bucket']['name'], record['s3']['object']['key'])


def lambda_event_handler(record):
    dt = datetime.datetime.strptime(record['time'], '%Y-%m-%dT%H:%M:%S%z')
    day = datetime.datetime(dt.year, dt.month, dt.day)
    run_query(Queries.kite_status_1d, day, day + datetime.timedelta(days=1))


lambda_event_source_handlers = {
    'aws:s3': lambda_s3_handler,
    'aws.events': lambda_event_handler,
}


s3_file_handlers = [
    (re.compile('^kite-metrics/firehose/kite_status/'), kite_status),
    (re.compile('^kite-metrics/firehose/client_events/'), index_metrics),
    (re.compile('^kite-backend-logs/firehose/server-logs/'), server_logs),
    (re.compile('^kite-metrics/athena-results/kite_status_1d/'), kite_status_1d),
    (re.compile('^kite-aws-billing-reports/'), aws_cost_reports),
]


async def run_pipeline(run_func):
    connections_var.set(Connections())
    try:
        await run_func()
    finally:
        es = connections_var.get().get('elasticsearch_async')
        if es:
            await es.transport.close()


def s3_handler(bucket, key):
    async def run_in_context(run_func):
        s3_var.set({'bucket': bucket, 'key': key})
        await run_pipeline(run_func)

    for pattern, handler in s3_file_handlers:
        if pattern.match('{}/{}'.format(bucket, key)):
            run_func, _ = handler(bucket=bucket, key=key)
            return asyncio.run(run_in_context(run_func))

    raise Exception("Unknown file, bucket={}, key={}".format(bucket, key))
