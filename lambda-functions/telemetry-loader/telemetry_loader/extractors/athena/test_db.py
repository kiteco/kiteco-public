import datetime

from telemetry_loader.extractors.athena.db import _get_partition_sql
from telemetry_loader.extractors.athena.db import get_prefix_bounds


def test_get_partition_sql():
    bounds = [datetime.datetime(2020, 2, 2, 13, 40), datetime.datetime(2020, 2, 2, 15, 0)]
    statements = _get_partition_sql('table_name', 's3://bucket/prefix/', get_prefix_bounds(*bounds))
    assert len(statements) == 6
    assert statements[0] == 'ALTER TABLE table_name ADD IF NOT EXISTS PARTITION (prefix=\'2020/02/02/12\') LOCATION \'s3://bucket/prefix/2020/02/02/12/\';'
    assert statements[3] == 'ALTER TABLE table_name ADD IF NOT EXISTS PARTITION (prefix=\'2020/02/02/15\') LOCATION \'s3://bucket/prefix/2020/02/02/15/\';'
