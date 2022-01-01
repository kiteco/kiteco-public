CREATE TABLE hubspot_rollup_{{ds_nodash}}
WITH (
  format='PARQUET',
  parquet_compression='SNAPPY',
  external_location = 's3://kite-metrics/athena/hubspot/intermediate/year={{execution_date.year}}/month={{execution_date.month}}/day={{execution_date.day}}/delta=0'
)
AS
WITH current AS (
  SELECT *
  FROM hubspot_intermediate hs
  WHERE (
    hs.delta=1 AND
    hs.year={{execution_date.year}} AND
    hs.month={{execution_date.month}} AND
    hs.day={{execution_date.day}}
  ) OR (
    hs.delta=0 AND
    hs.year={{(execution_date - macros.timedelta(days=1)).year}} AND
    hs.month={{(execution_date - macros.timedelta(days=1)).month}} AND
    hs.day={{(execution_date - macros.timedelta(days=1)).day}}
  )
),
scalar_aggs AS (
  SELECT
    {% for prop in params.scalar_props %}
      {%- if prop.sql.agg == 'latest' -%}
        coalesce(max_by({{prop.name}}, delta)) {{prop.name}},
      {%- elif prop.sql.delta_field -%}
        {{ prop.sql.agg }}(coalesce({{prop.name}}, {{prop.sql.delta_field}})) {{prop.name}},
      {%- else -%}
         {{ prop.sql.agg }}({{prop.name}}) {{prop.name}},
      {%- endif -%}
    {% endfor %}
    current.userid
  FROM current
  GROUP BY current.userid
)
SELECT scalar_aggs.userid,
    {% for prop in params.scalar_props %}
      {%- if prop.sql.agg_days -%}

        scalar_aggs.{{prop.name}} - coalesce(scalar_diff_{{ prop.sql.agg_days}}d.{{ prop.sql.delta_field or prop.name }}, 0) {{prop.name}}
      {%- else -%}
        scalar_aggs.{{prop.name}} {{prop.name}}
      {%- endif -%}
      {%- if (not loop.last) or params.map_props %},{% endif %}
    {% endfor %}
    {% for prop in params.map_props %}
      {{prop.name}}_aggs.value {{prop.name}}
      {%- if not loop.last %},{% endif %}
    {% endfor %}
FROM scalar_aggs
{% for tbl in params.scalar_time_rollups %}
LEFT JOIN hubspot_intermediate scalar_diff_{{tbl}}d
       ON scalar_aggs.userid = scalar_diff_{{tbl}}d.userid
      AND scalar_diff_{{tbl}}d.delta = 1
      AND scalar_diff_{{tbl}}d.year={{(execution_date - macros.timedelta(days=tbl)).year}}
      AND scalar_diff_{{tbl}}d.month={{(execution_date - macros.timedelta(days=tbl)).month}}
      AND scalar_diff_{{tbl}}d.day={{(execution_date - macros.timedelta(days=tbl)).day}}
{% endfor %}
{% for prop in params.map_props %}
LEFT JOIN (
  SELECT userid, transform_values(multimap_agg(k, v), (inner_k, inner_v) -> reduce(inner_v, cast(0 as bigint), (s, x) -> s + x, (s) -> s)) value
  FROM (
    SELECT userid, k, v
    FROM current
    CROSS JOIN unnest(coalesce({{prop.name}}, {{prop.sql.delta_field}})) as t(k, v)
    UNION ALL
    SELECT userid, k, v * -1
    FROM hubspot_intermediate
    CROSS JOIN unnest({{prop.sql.delta_field}}) as t(k, v)
    WHERE hubspot_intermediate.delta = 1
      AND hubspot_intermediate.year={{(execution_date - macros.timedelta(days=prop.sql.agg_days)).year}}
      AND hubspot_intermediate.month={{(execution_date - macros.timedelta(days=prop.sql.agg_days)).month}}
      AND hubspot_intermediate.day={{(execution_date - macros.timedelta(days=prop.sql.agg_days)).day}}
  )
  GROUP BY userid
) {{prop.name}}_aggs
      ON scalar_aggs.userid={{prop.name}}_aggs.userid
{% endfor %}