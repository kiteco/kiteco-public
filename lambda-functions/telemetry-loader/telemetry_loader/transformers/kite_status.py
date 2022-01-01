import datetime
import hashlib

from telemetry_loader.config import s3_var
from telemetry_loader.streams.core import pipe
from telemetry_loader.streams.pipes import json_pipe
from telemetry_loader.transformers.utils import get_index_shard
from telemetry_loader.transformers.utils import parse_date_from_key


@pipe
async def transform_elastic_kite_status_1d(it):
    async for rec in it:
        if rec['event'] != 'kite_status':
            continue
        rec_id = hashlib.md5('{}::{}'.format(rec.get('userId', ''), rec['end_time'].strftime('%Y/%m/%d'))
                             .encode('utf8')).hexdigest()
        rec['timestamp'] = rec['end_time']
        yield {'_index': 'kite_status_daily', '_id': rec_id, '_source': rec}


@pipe
def transform_mixpanel_kite_status_1d(rec):
    rec = rec.copy()
    rec['user_id'] = rec.pop('userId', '')
    rec['_group'] = 'firehose/kite_status/{}/'.format(rec['end_time'].strftime('%Y/%m/%d'))
    rec['time'] = int(rec['end_time'].timestamp())
    rec['start_time'] = int(rec['start_time'].timestamp())
    rec['end_time'] = int(rec['end_time'].timestamp())
    rec['_version'] = 0
    rec['source'] = 'kited'
    rec['name'] = rec.pop('event')
    return rec


INDEX_PREFIX = 'kite_status'
INDEX_GRANULARITY = datetime.timedelta(days=10)


def scrub(a_dict):
    if '' in a_dict:
        del a_dict['']
    for k, v in a_dict.items():
        if isinstance(v, dict):
            scrub(v)


languages = ['python', 'go', 'javascript']


@json_pipe()
async def transform_elastic_kite_status(it):
    dat = parse_date_from_key(s3_var.get()['key'])
    index_date_suffix = get_index_shard(dat, INDEX_GRANULARITY)
    async for doc in it:
        if doc.get('event') != 'kite_status':
            continue

        if 'messageId' not in doc:
            continue

        if 'properties' not in doc:
            continue

        index_active_str = 'active'
        if sum(doc['properties'].get('{}_events'.format(lang), 0) for lang in languages) == 0:
            continue

        index_name = '{}_{}_{}'.format(
            INDEX_PREFIX,
            index_active_str,
            index_date_suffix)

        scrub(doc)
        for field in ['originalTimestamp']:
            if field in doc:
                del doc[field]

        for field in ['repo_stats', 'receivedAt', 'sentAt', 'sent_at']:
            if field in doc['properties']:
                del doc['properties'][field]

        for field in ['cpu_samples_list', 'active_cpu_samples_list']:
            if not doc['properties'].get(field):
                continue
            p = field.split('_')[:-2]
            new_field = '_'.join(['max'] + p)
            doc['properties'][new_field] = max(map(float, doc['properties'][field]))

        # Next block is for backcompatibilty only
        # can be removed once the content of the PR https://github.com/kiteco/kiteco/pull/10638/ has been released to
        # most of our users
        for field in ['cpu_samples', 'active_cpu_samples']:
            if field in doc['properties']:
                samples_str = doc['properties'].pop(field)
                if len(samples_str) == 0:
                    continue
                p = field.split('_')[:-1]
                new_field = '_'.join(['max'] + p)
                doc['properties'][new_field] = max(map(float, samples_str.split(',')))

        doc['payload_size'] = len(doc)
        yield {'_index': index_name, '_id': doc['messageId'], '_source': doc}
