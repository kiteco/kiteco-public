CREATE EXTERNAL TABLE `youtube_queries`(
    tagname	string,
    count bigint,
    query string,
    seed boolean,
    generation int
)
ROW FORMAT SERDE
  'org.openx.data.jsonserde.JsonSerDe'
STORED AS INPUTFORMAT
  'org.apache.hadoop.mapred.TextInputFormat'
OUTPUTFORMAT
  'org.apache.hadoop.hive.ql.io.HiveIgnoreKeyTextOutputFormat'
LOCATION
  's3://kite-youtube-data/search_queries/'
