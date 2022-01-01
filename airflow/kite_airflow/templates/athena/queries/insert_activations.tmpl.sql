INSERT INTO activations
WITH new_activations as (
  SELECT coalesce(properties__user_id, properties__anonymous_id) userid,
         min(from_iso8601_timestamp(timestamp)) activation_date,
         to_unixtime(min(from_iso8601_timestamp(timestamp))) activation_timestamp
  FROM kite_status_normalized
  WHERE year = {{execution_date.year}} AND month = {{execution_date.month}} AND day = {{execution_date.day}}
    AND (event='ast_node_resolved' OR event='anon_supported_file_edited')
  GROUP BY coalesce(properties__user_id, properties__anonymous_id)
)
SELECT new_activations.userid,
       new_activations.activation_timestamp,
       day(new_activations.activation_date) day,
       year(new_activations.activation_date) year,
       month(new_activations.activation_date) month
FROM activations
RIGHT OUTER JOIN new_activations ON activations.userid=new_activations.userid
WHERE new_activations.activation_timestamp < activations.activation_timestamp
   OR activations.userid IS NULL