import json
import base64
import datetime
import gzip
from io import BytesIO

from telemetry_loader.config import s3_var
from telemetry_loader.streams.pipes import json_pipe
from telemetry_loader.transformers.utils import parse_date_from_key
from telemetry_loader.transformers.utils import get_index_shard


INDEX_GRANULARITY = datetime.timedelta(days=10)


def resolve_dotted_path(doc, path):
    container = doc
    field_name = path
    while '.' in field_name:
        container_name, field_name = path.split('.', 1)
        if container_name not in container:
            return None, None
        container = container[container_name]

    if field_name in container:
        return container, field_name

    return None, None


@json_pipe()
async def transform_elastic_index_metrics(it):
    dat = parse_date_from_key(s3_var.get()['key'])
    index_date_suffix = get_index_shard(dat, INDEX_GRANULARITY)

    async for doc in it:
        if 'messageId' not in doc:
            continue

        if 'properties' not in doc:
            continue

        event = doc.get('event')
        if event == 'Index Build':
            index_prefix = 'index_build'
        elif event == 'Completion Stats':
            index_prefix = 'completions_selected'
        else:
            continue

        index_name = '{}_{}'.format(index_prefix, index_date_suffix)

        for field in ['originalTimestamp']:
            if field in doc:
                del doc[field]

        for field in ['repo_stats', 'receivedAt', 'sentAt', 'sent_at', 'parse_info.parse_errors']:
            container, field_name = resolve_dotted_path(doc['properties'], field)
            if container:
                del container[field_name]

        for field in ['cpu_info.sum', 'lexical_metrics.score']:
            container, field_name = resolve_dotted_path(doc['properties'], field)
            if container:
                container[field_name] = float(container[field_name])

        for field in ['completion_stats']:
            if field in doc['properties']:
                # completions_stats is an encoded list
                data = doc['properties'][field]
                data = base64.b64decode(data)
                data = gzip.GzipFile(fileobj=BytesIO(data)).read()
                data = json.loads(data)
                del doc['properties'][field]
                # create one document per completion stat
                i = 0
                for stat in data:
                    i += 1
                    elem = doc
                    for key in stat:
                        elem['properties'][key] = stat[key]
                    yield {
                        '_index': index_name,
                        '_id': doc['messageId'] + "-" + str(i),
                        '_source': elem
                    }

            else:
                yield {
                    '_index': index_name,
                    '_id': doc['messageId'],
                    '_source': doc
                }
