CREATE EXTERNAL TABLE `kite_link_stats_youtube_channels`(
    id string,
    name string,
    last_updated string,
    is_backfilled boolean,
    last_backfill_until string
)
ROW FORMAT SERDE
  'org.openx.data.jsonserde.JsonSerDe'
STORED AS INPUTFORMAT
  'org.apache.hadoop.mapred.TextInputFormat'
OUTPUTFORMAT
  'org.apache.hadoop.hive.ql.io.HiveIgnoreKeyTextOutputFormat'
LOCATION
  '{{ params.data_location }}'
