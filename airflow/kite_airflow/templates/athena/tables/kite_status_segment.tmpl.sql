CREATE EXTERNAL TABLE `kite_status_segment`(
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
  's3://kite-metrics/segment-logs/XXXXXXX'
TBLPROPERTIES (
  'projection.enabled'='true',
  'projection.prefix.interval'='XXXXXXX',
  'projection.prefix.range'='XXXXXXX,XXXXXXX',
  'projection.prefix.type'='integer',
  'storage.location.template'='s3://kite-metrics/segment-logs/XXXXXXX/${prefix}'
)
