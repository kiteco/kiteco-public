import datetime


def parse_date_from_key(key):
    parts = key.split('/')
    if parts[0] == "firehose":
        # firehose/{stream_name}/{yyyy}/{mm}/{dd}/{hh}/{random_filename}.gz
        yyyy, mm, dd = parts[2:5]
        return datetime.date(year=int(yyyy), month=int(mm), day=int(dd))
    if parts[0] == "segment-logs":
        # segment-logs/{source_id}/{unix_ts_ms}/{random_filename}.gz
        unix_ts_ms = int(parts[2])
        unix_ts = unix_ts_ms / 1000
        return datetime.datetime.utcfromtimestamp(unix_ts).date()
    raise ValueError("unhandled S3 prefix")


def get_index_shard(date, granularity, epoch=datetime.date(1970, 1, 1)):
    rounded = epoch + (date - epoch) // granularity * granularity
    return rounded.isoformat()
