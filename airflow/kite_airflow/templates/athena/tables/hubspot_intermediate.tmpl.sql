CREATE EXTERNAL TABLE IF NOT EXISTS `hubspot_intermediate` (
  userid string,
  {% for prop in params.props if prop.sql.type %}
  {{ prop.name }} {{ prop.sql.type }}{% if not loop.last %},{% endif %}
  {% endfor %}
)
PARTITIONED BY (
  `year` int,
  `month` int,
  `day` int,
  `delta` int
)
STORED AS PARQUET
LOCATION 's3://kite-metrics/athena/hubspot/intermediate/'
TBLPROPERTIES (
  'projection.enabled'='true',
  'projection.year.type'='integer',
  'projection.year.range'='2010,2100',
  'projection.month.type'='integer',
  'projection.month.range'='1,12',
  'projection.day.type'='integer',
  'projection.day.range'='1,31',
  'projection.delta.type'='integer',
  'projection.delta.range'='0,1',
  'storage.location.template'='s3://kite-metrics/athena/hubspot/intermediate/year=${year}/month=${month}/day=${day}/delta=${delta}'
);
