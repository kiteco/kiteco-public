INSERT INTO
  {{ params.table_name }} (userid, timestamp)
SELECT
  DISTINCT userid,
  timestamp
FROM
  kite_status_normalized
WHERE
  (
    {% for language in params.languages %}
      properties__{{language}}_events > 0 {% if not loop.last -%} OR {%- endif -%}
    {% endfor %}
  )
  AND regexp_like(userid, '\p{Cc}') = FALSE -- filter user id's that contains null
  AND regexp_replace(kite_status_normalized.userid, '\x{00}') != '' -- filter user id's were only null bytes
  AND year >= {{ execution_date.year }}
  AND month >= {{ execution_date.month }}
  AND day >= {{ execution_date.day }}
  AND hour > {{ execution_date.hour }}
;
