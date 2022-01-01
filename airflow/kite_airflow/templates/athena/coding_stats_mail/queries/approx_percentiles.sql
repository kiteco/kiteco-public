-- Week range is Sunday - Saturday
{% set start_date=execution_date %}
{% set end_date=execution_date.add(days=6) %}

WITH any_edit AS (
  SELECT
    userid,
    CAST(
      COUNT_IF(
        {% for language in params.languages %}
          properties__{{language}}_edit > 0 {% if not loop.last -%} OR {%- endif -%}
        {% endfor %}
      ) AS double
    ) / 6 AS edits -- dividing by 6 because events get reported after every 10 minutes
  FROM
    kite_status_normalized
  WHERE
  (
    year > {{ start_date.year }}
    OR (year = {{ start_date.year }} AND month > {{ start_date.month }})
    OR (year = {{ start_date.year }} AND month = {{ start_date.month }} AND day >= {{ start_date.day }})
  )
  AND (
    year < {{ end_date.year }}
    OR (year = {{ end_date.year }} AND month < {{ end_date.month }})
    OR (year = {{ end_date.year }} AND month = {{ end_date.month }} AND day <= {{ end_date.day }})
  )
  GROUP BY
    userid
)
SELECT
  {% for i in range(1, 100) %}
    approx_percentile(edits, {{i/100}}) AS pct_{{i}} {% if i < 99 -%} , {%- endif -%}
  {% endfor %}
FROM
  any_edit
WHERE
  edits > 0;
