-- Calculating coding stats of provided NUM_OF_WEEKS.
-- Week range is Sunday - Saturday
{% set start_date=execution_date.subtract(days=(7 * (params.num_of_weeks - 1))) %}
{% set end_date=execution_date.add(days=6) %}

WITH coding_stats AS (
  SELECT
    userid,
    date_diff(
      'day',
      from_iso8601_timestamp(timestamp),
      CAST('{{end_date.to_date_string()}}' AS timestamp)
    ) / 7 AS week,
    SUM(
      {% for language in params.languages %}
        COALESCE(properties__{{language}}_completions_num_selected, 0)  {% if not loop.last -%} + {%- endif -%}
      {% endfor %}
    ) AS completions_selected,
    CAST(
      COUNT_IF(
        {% for language in params.languages %}
          properties__{{language}}_edit > 0 {% if not loop.last -%} OR {%- endif -%}
        {% endfor %}
      ) AS double
    ) / 6 AS coding_hours,
    CAST(
      COUNT_IF(properties__python_edit > 0) AS double
    ) / 6 AS python_hours
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
    AND event = 'kite_status'
    AND regexp_like(kite_status_normalized.userid, '\\p{Cc}') = FALSE -- filter user id's that contains null
    AND regexp_replace(kite_status_normalized.userid, '\x{00}') != '' -- filter user id's were only null bytes
  GROUP BY
    1,
    2
)
SELECT
  coding_stats.userid,
  map_agg(week, completions_selected) AS completions_selected,
  map_agg(week, coding_hours) AS coding_hours,
  map_agg(week, python_hours) AS python_hours,
  reduce(
    array_agg(
      from_iso8601_timestamp({{ params.table_daily_active_users }}.timestamp)
      ORDER BY
        {{ params.table_daily_active_users }}.timestamp DESC
    ),
    0,
    (acc, current) -> if(
      date_diff('day', current, CAST('{{end_date.to_date_string()}}' AS timestamp)) / 7 - acc < 1,
      date_diff('day', current, CAST('{{end_date.to_date_string()}}' AS timestamp)) / 7 + 1,
      acc
    ),
    acc -> acc
  ) AS streak,
  COUNT(
    DISTINCT date_diff(
      'day',
      from_iso8601_timestamp({{ params.table_daily_active_users }}.timestamp),
      CAST('{{end_date.to_date_string()}}' AS timestamp)
    ) / 7
  ) AS total_weeks
FROM
  coding_stats
  LEFT OUTER JOIN {{ params.table_daily_active_users }} ON coding_stats.userid = {{ params.table_daily_active_users }}.userid
GROUP BY
  coding_stats.userid
;
