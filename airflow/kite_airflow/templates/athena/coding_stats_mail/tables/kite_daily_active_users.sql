CREATE EXTERNAL TABLE IF NOT EXISTS `{{params.table_name}}`(
  userid string,
  timestamp string
)
STORED AS PARQUET
LOCATION
  '{{ params.data_location }}'
