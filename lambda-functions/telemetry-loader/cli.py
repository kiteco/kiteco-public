import base64
import json
import click
import click_log
import datetime
import logging
import urllib
import boto3

from telemetry_loader.handler import s3_handler
from telemetry_loader.extractors.athena.db import Queries, run_query


s3 = boto3.client('s3')
lambda_client = boto3.client('lambda')
dl_queue_url = 'https://sqs.us-east-1.amazonaws.com/XXXXXXX/telemetry-loader-elastic-dl'

logger = logging.getLogger('telemetry_loader')
click_log.basic_config(logger)


@click.group()
def cli():
    pass


@cli.command()
@click.argument('s3_path')
@click.option('--profile/--no-profile', default=False)
@click_log.simple_verbosity_option(logger)
def load(s3_path, profile):
    parsed = urllib.parse.urlparse(s3_path)
    if profile:
        import cProfile
        import pstats
        import io
        pr = cProfile.Profile()
        pr.enable()
    s3_handler(parsed.netloc, parsed.path[1:])
    if profile:
        pr.disable()
        s = io.StringIO()
        ps = pstats.Stats(pr, stream=s).sort_stats('cumtime')
        ps.print_stats(.05)
        click.echo(s.getvalue())


@cli.command()
@click.argument('s3_path')
@click_log.simple_verbosity_option(logger)
def invoke(s3_path):
    parsed = urllib.parse.urlparse(s3_path)
    keys = []
    if parsed.path[1:].endswith('/'):
        paginator = s3.get_paginator('list_objects_v2')
        result = paginator.paginate(Bucket=parsed.netloc, Delimiter='/', Prefix=parsed.path[1:])
        for item in result:
            keys.extend([c['Key'] for c in item['Contents']])
    else:
        keys.append(parsed.path[1:])

    for key in keys:
        logger.info("Running {}".format(key))
        payload = json.dumps({'Records': [
            {'eventSource': 'aws:s3', 's3': {'bucket': {'name': parsed.netloc}, 'object': {'key': key}}}
        ]})
        res = lambda_client.invoke(FunctionName='telemetry-loader-elastic', LogType='Tail', Payload=payload)
        click.echo(res)


@cli.command()
@click_log.simple_verbosity_option(logger)
@click.argument('query_name')
@click.argument('date')
@click.option('--prod/--no-prod', default=False)
def run_athena_query(query_name, date, prod):
    if '-' in date:
        start, end = (datetime.datetime.strptime(d, '%m/%d/%Y') for d in date.split('-'))
    else:
        start = datetime.datetime.strptime(date, '%m/%d/%Y')
        end = start + datetime.timedelta(days=1)

    run_query(getattr(Queries, query_name), start, end, prod=prod)


@cli.command()
@click_log.simple_verbosity_option(logger)
def process_dl_queue():
    client = boto3.client('sqs')
    while True:
        msgs = client.receive_message(QueueUrl=dl_queue_url, MaxNumberOfMessages=1, VisibilityTimeout=1200)
        if not msgs['Messages']:
            return
        msg = msgs['Messages'][0]

        msg_body = json.loads(msg['Body'])

        if msg_body['Records'][0].get('eventSource') == 'aws:s3' and msg_body['Records'][0]['s3']['bucket']['name'] == 'kite-backend-logs':
            client.delete_message(QueueUrl=dl_queue_url, ReceiptHandle=msg['ReceiptHandle'])
            continue

        click.echo("message={}".format(msg['Body']))

        if msg_body['Records'][0].get('eventSource') == 'aws:s3' and msg_body['Records'][0]['s3']['object']['key'].startswith('athena-results'):
            client.delete_message(QueueUrl=dl_queue_url, ReceiptHandle=msg['ReceiptHandle'])
            continue

        # value = click.prompt('Try re-running?', type=bool)
        if True:
            res = lambda_client.invoke(FunctionName='telemetry-loader-elastic', LogType='Tail', Payload=msg['Body'])
            if not res.get('FunctionError'):
                client.delete_message(QueueUrl=dl_queue_url, ReceiptHandle=msg['ReceiptHandle'])
                continue
            click.echo("log={}".format(base64.b64decode(res['LogResult'])))
        continue

        value = click.prompt('Delete from queue?', type=bool)
        if not value:
            return
        client.delete_message(QueueUrl=dl_queue_url, ReceiptHandle=msg['ReceiptHandle'])


if __name__ == '__main__':
    cli()
