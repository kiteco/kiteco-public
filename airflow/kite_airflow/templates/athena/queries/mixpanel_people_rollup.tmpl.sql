CREATE TABLE mixpanel_people_rollup_{{ds_nodash}}
WITH (
  format='PARQUET',
  parquet_compression='SNAPPY',
  external_location = 's3://kite-metrics/mixpanel/people/rollups/year={{execution_date.year}}/month={{execution_date.month}}/day={{execution_date.day}}/'
)
AS
WITH candidates AS (
  SELECT {% for key in params.schema|sort %}{{ key }}{% if not loop.last %}, {% endif %}{% endfor %}
  FROM mixpanel_people_raw
  {% if prev_execution_date_success %}WHERE year > {{ prev_execution_date_success.year }} OR (year = {{ prev_execution_date_success.year }} AND month > {{ prev_execution_date_success.month }} ) OR  (year = {{ prev_execution_date_success.year }} AND month = {{ prev_execution_date_success.month }} AND day > {{ prev_execution_date_success.day }}) {% endif %}
  UNION ALL
  SELECT {% for key in params.schema|sort %}{{ key }}{% if not loop.last %}, {% endif %}{% endfor %}
  FROM mixpanel_people
)
SELECT
  distinct_id,
{% for key in params.schema if key != 'distinct_id' %}
  max_by({{ key}}, time) {{ key }}{% if not loop.last %},{% endif %}
{% endfor %}
FROM candidates
GROUP BY distinct_id
