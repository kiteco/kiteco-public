CREATE TABLE hubspot_delta_{{ds_nodash}}
WITH (
  format='PARQUET',
  parquet_compression='SNAPPY',
  external_location = 's3://kite-metrics/athena/hubspot/intermediate/year={{execution_date.year}}/month={{execution_date.month}}/day={{execution_date.day}}/delta=1'
)
AS
SELECT
{% for prop in params.props %}
  {% if prop.sql.delta -%}
    CAST({{prop.sql.delta}} AS {{prop.sql.type}}) {{prop.name}},
  {%- endif -%}
  {% if prop.sql.map_delta -%}
    transform_values(multimap_agg({{prop.sql.map_delta}}), (k, v) -> reduce(v, 0, (s, x) -> s + x, (s) -> s)) {{prop.name}},
  {%- endif -%}
{% endfor %}
       userid
FROM kite_status_normalized
WHERE year={{execution_date.year}} AND month={{execution_date.month}} AND day={{execution_date.day}}
      AND regexp_like(userid, '^[0-9]+$')
GROUP BY userid