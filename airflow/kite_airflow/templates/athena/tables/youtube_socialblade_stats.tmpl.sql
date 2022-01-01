CREATE EXTERNAL TABLE `youtube_socialblade_stats`(
    id string,
    timestamp timestamp,
    success boolean,
    monthlyViews string
)
ROW FORMAT SERDE
  'org.openx.data.jsonserde.JsonSerDe'
STORED AS INPUTFORMAT
  'org.apache.hadoop.mapred.TextInputFormat'
OUTPUTFORMAT
  'org.apache.hadoop.hive.ql.io.HiveIgnoreKeyTextOutputFormat'
LOCATION
  's3://kite-youtube-data/socialblade_stats/'
