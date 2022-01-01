CREATE TABLE kite_status_1d_{{params.key}}_{{ds_nodash}}_json
WITH (
  format='JSON',
  external_location = 's3://kite-metrics/athena/kite_status_1d_{{params.key}}/json/{{ds}}'
)
AS
SELECT *
FROM kite_status_1d_{{params.key}}_{{ds_nodash}}
WHERE ({% for lang in params.languages %}{{lang}}_events > 0{% if not loop.last %} OR {% endif %}{% endfor %})
  AND year = {{execution_date.year}}
  AND month = {{execution_date.month}}
  AND day = {{execution_date.day}}
