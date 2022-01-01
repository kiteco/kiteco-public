CREATE EXTERNAL TABLE `youtube_channel_details`(
    id string,
    forUsername string,
    snippet struct<
      title: string,
      customUrl: string
    >,
    statistics struct<
      viewCount: string,
      subscriberCount: string,
      videoCount: string
    >
)
ROW FORMAT SERDE
  'org.openx.data.jsonserde.JsonSerDe'
STORED AS INPUTFORMAT
  'org.apache.hadoop.mapred.TextInputFormat'
OUTPUTFORMAT
  'org.apache.hadoop.hive.ql.io.HiveIgnoreKeyTextOutputFormat'
LOCATION
  's3://kite-youtube-data/channel_details/'
