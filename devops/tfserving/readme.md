# Provisionning script for tfServing instance

#### This script requires to be connected to **AWS VPN** to work.  

TfServing is a container provided by tensorflow to easily serve a NN model remotely. 

We use nvidia-docker (now integrated in docker command with `--gpus` arg) to be able to use the GPU from inside the container. 

This script currently only maintain 1 instance in GCP (so each change/apply destroy the instance first before rebuilding it). 

The instance is accessible on all port through AWS VPN and exposes the ports 8500 and 8501 on a static external IP mapped to tfserving-dev.kite.com.

## S3 Access

This instance can access and list bucket for the bucket kite-data on s3. That allows to access the models directly from the instance. 

The credentials are using the IAM user `instance_role_tfserving`. 

TfServing container is also configured to use the same AWS credentials so the config file for the model can have a s3 path for the `base_path` of a model.


## Configuration 

By default, the file `default_model_config_list.txt` is used. The path for another file can be used by setting the variable `model_config_list_path` when calling terraform. 

## Monitoring

The instance runs a metricbeat process that gets data from 2 prometheus exporter and the system metrics:
- TfServing monitoring (on port 8501), enable with the file `monitoring_config.txt` passed when we spin up the tfServing container
- GPU monitoring, using the docker container `mindprince/nvidia_gpu_prometheus_exporter:0.1` on the port 9445
- System metric of the GCP instance (CPU usage, top 5 process, etc.)

These metrics are sent to our elasticsearch instance with the tag tfserving-dev-gcp and can be found easily in the Explorer with the search `fields.node_name:tfserving-dev-gcp`

If you want only the metrics coming from prometheus use the search : 
`fields.node_name:tfserving-dev-gcp and prometheus.labels.job:prometheus`

If you want only tfServing metrics : 
`fields.node_name:tfserving-dev-gcp and prometheus.labels.instance:"localhost:8501"` 

If you want only GPU metrics :
`fields.node_name:tfserving-dev-gcp and prometheus.labels.instance:"localhost:9445"`