CREATE TABLE cio_profile_attrs_{{ds_nodash}}
WITH (
  format='JSON',
  external_location = 's3://kite-metrics/athena/cio_profile_attrs/{{ds}}'
)
AS
WITH current AS (
  SELECT *
  FROM hubspot_intermediate
  WHERE year = {{execution_date.year}}
    AND month = {{execution_date.month}}
    AND day = {{execution_date.day}}
    AND delta=0
)
SELECT
  current.userid id,
{% for prop in params.props -%}
  subquery_{{prop}}.value {{prop}}
  {%- if not loop.last -%},{% endif %}
{%- endfor %}
FROM current
{% for prop in params.props %}
LEFT JOIN (
  SELECT userid, max_by(k, v) value
  FROM current
  CROSS JOIN unnest(user_data_{{prop}}) AS t(k, v)
  GROUP BY userid
) subquery_{{ prop }}
  ON current.userid = subquery_{{ prop }}.userid
{%- endfor %}
