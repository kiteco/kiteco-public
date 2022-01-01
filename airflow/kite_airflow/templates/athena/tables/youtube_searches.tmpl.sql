CREATE EXTERNAL TABLE `youtube_searches`(
    query string,
    query_hash string,
    timestamp string,
    total int,
    unique int
)
ROW FORMAT SERDE
  'org.openx.data.jsonserde.JsonSerDe'
STORED AS INPUTFORMAT
  'org.apache.hadoop.mapred.TextInputFormat'
OUTPUTFORMAT
  'org.apache.hadoop.hive.ql.io.HiveIgnoreKeyTextOutputFormat'
LOCATION
  's3://kite-youtube-data/searches/'
