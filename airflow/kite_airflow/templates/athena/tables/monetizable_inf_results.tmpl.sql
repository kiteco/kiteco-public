CREATE EXTERNAL TABLE monetizable_inf_results_{{ds_nodash}} (
  userid string,
  score double,
  model_version string,
  timestamp bigint
)
ROW FORMAT SERDE
  'org.openx.data.jsonserde.JsonSerDe'
LOCATION
  's3://{{params.bucket}}/monetizable/inf_results/{{ds_nodash}}/'
