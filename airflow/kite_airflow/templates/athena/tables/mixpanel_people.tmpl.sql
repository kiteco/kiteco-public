{% macro struct(dct) -%}
  struct<
    {% for key, value in dct.items() %}
      {{ key }}: {% if value is mapping %}{{ struct(value) }}{% else %}{{ value }}{% endif %}{% if not loop.last %},{% endif %}
    {% endfor %}
  >
{%- endmacro %}

CREATE EXTERNAL TABLE `{{ params.table_name }}` (
  {% for key, value in params.schema.items() %}
    {{ key }} {% if value is mapping %}{{ struct(value) }}{% else %}{{ value }}{% endif %}{% if not loop.last %},{% endif %}
  {% endfor %}
)
{% if params.partitioned %}
PARTITIONED BY (
  `year` int,
  `month` int,
  `day` int
)
{% endif %}
{% if params.json %}
ROW FORMAT SERDE
  'org.openx.data.jsonserde.JsonSerDe'
STORED AS INPUTFORMAT
  'org.apache.hadoop.mapred.TextInputFormat'
OUTPUTFORMAT
  'org.apache.hadoop.hive.ql.io.HiveIgnoreKeyTextOutputFormat'
LOCATION
  's3://kite-metrics/{{ params.s3_prefix }}'
TBLPROPERTIES (
  'projection.enabled'='true',
  'projection.year.type'='integer',
  'projection.year.range'='2020,2100',
  'projection.month.type'='integer',
  'projection.month.range'='1,12',
  'projection.day.type'='integer',
  'projection.day.range'='1,31',
  'storage.location.template'='s3://kite-metrics/{{ params.s3_prefix }}/year=${year}/month=${month}/day=${day}'
)
{% else %}
STORED AS PARQUET
LOCATION
  's3://kite-metrics/{{ params.s3_prefix }}'
{% endif %}
