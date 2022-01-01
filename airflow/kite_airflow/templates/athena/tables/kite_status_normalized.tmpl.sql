CREATE EXTERNAL TABLE `kite_status_normalized`(
  {% for key, value in params.schema['properties'].items()|sort if key != "properties" and value.get("kite_status_normalized") != False %}
  `{{ key }}` {{ value.type|safe }},
  {% endfor %}

  {% for key, value in params.schema['properties']['properties']['properties'].items()|sort if value.get("kite_status_normalized") != False %}
  `properties__{{ key }}` {{ value.type|safe }},
  {% endfor %}

  `maxmind__country_name` string,
  `maxmind__country_iso_code` string,
  `maxmind__subdivision_1_name` string,
  `maxmind__city_name` string,
  `maxmind__time_zone` string,
  `monetizable_score` double,
  `monetizable_model_version` string
)
PARTITIONED BY (
  `year` int,
  `month` int,
  `day` int,
  `hour` int
)
STORED AS PARQUET
LOCATION 's3://kite-metrics/athena/kite_status_normalized/'
TBLPROPERTIES (
  'projection.enabled'='true',
  'projection.year.type'='integer',
  'projection.year.range'='2010,2100',
  'projection.month.type'='integer',
  'projection.month.range'='1,12',
  'projection.day.type'='integer',
  'projection.day.range'='1,31',
  'projection.hour.type'='integer',
  'projection.hour.range'='0,23',
  'storage.location.template'='s3://kite-metrics/athena/kite_status_normalized/year=${year}/month=${month}/day=${day}/hour=${hour}'
);
