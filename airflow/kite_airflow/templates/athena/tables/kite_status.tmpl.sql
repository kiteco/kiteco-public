CREATE EXTERNAL TABLE `kite_status`(
  {%- for key, value in params.schema['properties'].items()|sort if key != "properties" and value.get("kite_status_normalized") != False %}
  `{{ key }}` {{ value.type|safe }},
  {%- endfor %}
  `properties` struct<
  {% for key, value in params.schema['properties']['properties']['properties'].items()|sort if value.get("kite_status_normalized") != False -%}
  {{ key }}:{{ value.type|safe }}{% if not loop.last %},{% endif %}
  {%- endfor %}
  >
)
PARTITIONED BY (
  `prefix` string)
ROW FORMAT SERDE
  'org.openx.data.jsonserde.JsonSerDe'
STORED AS INPUTFORMAT
  'org.apache.hadoop.mapred.TextInputFormat'
OUTPUTFORMAT
  'org.apache.hadoop.hive.ql.io.HiveIgnoreKeyTextOutputFormat'
LOCATION
  's3://kite-metrics/firehose/kite_status'
TBLPROPERTIES (
  'projection.enabled'='true',
  'projection.prefix.format'='yyyy/MM/dd/HH',
  'projection.prefix.interval'='1',
  'projection.prefix.interval.unit'='HOURS',
  'projection.prefix.range'='2018/01/01/00,NOW',
  'projection.prefix.type'='date',
  'storage.location.template'='s3://kite-metrics/firehose/kite_status/${prefix}'
)