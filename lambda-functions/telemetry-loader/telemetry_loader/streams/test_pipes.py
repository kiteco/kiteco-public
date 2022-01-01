import csv
import pytest

from telemetry_loader.streams.core import consume
from telemetry_loader.streams.core import stream
from telemetry_loader.streams.pipes import csv_pipe
from telemetry_loader.streams.pipes import json_pipe


pytestmark = pytest.mark.asyncio


async def test_dict_reader():
    run, _ = stream([b'f1,f2\n', b'v1.1,v1.2\n', b'v2.1,v2.2\n']) | csv_pipe(csv.DictReader) | consume
    assert [dict(line) for line in await run()] == [{'f1': 'v1.1', 'f2': 'v1.2'}, {'f1': 'v2.1', 'f2': 'v2.2'}]


async def test_json_pipe():
    run, _ = stream([b'{"a": 1}\n', b'{"b": 2}\n']) | json_pipe() | consume
    assert [line for line in await run()] == [{'a': 1}, {'b': 2}]
