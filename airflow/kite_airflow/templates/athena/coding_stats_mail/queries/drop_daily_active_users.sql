{% if prev_execution_date_success == None -%}
  DROP TABLE IF EXISTS {{ params.table_name }};
{%- else -%}
  -- void query which prevents ERROR: Parameter validation failed which occurs due to empty file that have no query
  SELECT *
  FROM {{ params.table_name }}
  LIMIT 0;
{%- endif -%}
