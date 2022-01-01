import csv
import datetime
import io
import json
import time

from telemetry_loader.streams.core import pipe


def infer_types(value):
    if isinstance(value, dict):
        return {k: infer_types(v) for k, v in value.items() if v != ''}
    if value == 'true':
        return True
    if value == 'false':
        return False
    if value.isdigit():
        return int(value)
    try:
        return float(value)
    except ValueError:
        pass
    try:
        return datetime.datetime.strptime(value, "%Y-%m-%dT%H:%M:%S.%f%z")
    except ValueError:
        pass
    return value


def csv_pipe(get_csv_reader=csv.DictReader, infer_types_fn=infer_types):
    @pipe
    async def _csv_parser(it):
        buffer = io.BytesIO()
        reader = get_csv_reader(io.TextIOWrapper(buffer))

        async for line in it:
            if isinstance(line, str):
                line = line.encode('utf8')

            if isinstance(reader, csv.DictReader) and not reader._fieldnames:
                buffer.write(line)
                buffer.seek(0)
                reader.fieldnames
                buffer.seek(0)
                buffer.truncate(0)
                continue

            buffer.write(line)
            buffer.seek(0)
            yield infer_types_fn(next(reader))
            buffer.seek(0)
            buffer.truncate(0)

    return _csv_parser


def json_pipe(json_lib=json):
    @pipe
    def _json_pipe(line):
        return json_lib.loads(line)
    return _json_pipe


def progress(log_fn, log_every='1s'):
    log_every_s = int(log_every[:-1])

    @pipe
    async def _progress_pipe(it):
        start = time.time()
        last = start
        lines = 0
        nbytes = 0
        async for line in it:
            lines += 1
            nbytes += len(line)
            yield line
            if (time.time() - last) > log_every_s:
                last = time.time()
                log_fn("{} lines ({} bytes): {} lines/sec".format(lines, nbytes, lines / (last - start)))
        else:
            try:
                log_fn("Completed: {} lines ({} bytes): {} lines/sec".format(lines, nbytes, lines / (last - start)))
            except Exception:
                pass
    return _progress_pipe


def head(count):
    @pipe
    async def _head(it):
        n = 0
        async for line in it:
            yield line
            n += 1
            if n == count:
                return
    return _head
