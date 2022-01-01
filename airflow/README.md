Kite Airflow
============

UI
-------------

Airflow is deploy to https://airflow.kite.dev. Requires VPN.

How to Deploy
-------------

Requirements:
 * AWS CLI
 * JQ (https://stedolan.github.io/jq/download/)
 * Docker

Deployment:
 * Login to AWS ECR: make docker_login
 * Deploy: make build deploy
 * Confirm Terraform deploy by type "yes"

To see deployment status:
 * make show_containers

Adding metrics to kite status 1d
--------------------------------

 * Ensure the field is in dags/files/kite_status.schema.yaml.
 * Add the aggregation to dags/templates/athena/queries/kite_status_1d.tmpl.sql.
 * Deploy.
 * Manually trigger the DAG "update_kite_status_schema": http://XXXXXXX:8080/admin/airflow/tree?dag_id=update_kite_status_schema
 * Let the kite_status_1d jobs run at their normally-scheduled time.
