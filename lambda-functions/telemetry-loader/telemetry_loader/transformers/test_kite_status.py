import json
import pytest

from telemetry_loader.config import s3_var
from telemetry_loader.transformers.kite_status import transform_elastic_kite_status
from telemetry_loader.streams.core import stream, consume


pytestmark = pytest.mark.asyncio


async def test_only_active():
    recs = [
        {'properties': {'python_events': 4, 'go_events': 2}},
        {'properties': {'python_events': 0, 'go_events': 0}},
        {'properties': {'python_events': 0, 'go_events': 0, 'javascript_events': 0}},
    ]

    def generate_messages(recs):
        for i, rec in enumerate(recs):
            rec['messageId'] = str(i)
            rec['event'] = 'kite_status'
            yield(json.dumps(rec).encode('utf8'))

    s3_var.set({'bucket': 'bucket', 'key': 'firehose/kite_status/2020/04/20/01/file.gz'})
    run, _ = stream(generate_messages(recs)) | transform_elastic_kite_status | consume
    assert [line['_source']['properties'] for line in await run()] == [{'python_events': 4, 'go_events': 2}]
