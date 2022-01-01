CREATE TABLE monetizable_scores_{{ds_nodash}}
WITH (
  format='PARQUET',
  parquet_compression='SNAPPY',
  external_location = 's3://{{params.bucket}}/monetizable/final_users/{{ds_nodash}}'
)
AS SELECT
  userid,
  max_by(score, timestamp) score,
  max_by(model_version, timestamp) model_version,
  max(timestamp) timestamp
FROM
  (
    SELECT userid, score, model_version, timestamp FROM monetizable_scores
    UNION ALL
    SELECT userid, score, model_version, {{ execution_date.int_timestamp }} FROM monetizable_inf_results_{{ds_nodash}}
  ) AS subq
GROUP BY userid
