# Services

## GKE Cluster Setup

Project: kite-dev
Network: kite-dev
Subnet: kite-dev-private-<region>
Private cluster: enabled
Workload Identity: enabled


Pick address ranges based on your region and cluster type:

Master address range:  10.201.<cluster_type_master_range>.<region_index * 14>/28
Service address range: 10.201.<cluster_svc_service_range + region_index>.0/24

cluster_type_master_range is:
 * prod: 101
 * cloud: 102

cluster_svc_service_range:
 * prod: 105
 * cloud: 155

region_index:
 * us-west1: 0

Master authorized networks:
  * VPN: 10.86.0.0/16

Install gtoken:

`cd gtoken && make REGION=<region> CLUSTER=<cluster>`

## GKE Node Pool Setup

GKE Metadata Server: enabled