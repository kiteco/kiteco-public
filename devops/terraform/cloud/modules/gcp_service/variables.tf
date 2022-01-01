variable "name" {
  type = string
}

variable "versions" {
  type = map(string)
}

variable "network" {
  type = string
}

variable "aws_acces_key_id" {
  type = string
}

variable "gcp_aws_secret_access_key" {
  type = string
}

variable "service_account_email" {
  type = string
}

variable "min_replicas" {
  type = number
}

variable "max_replicas" {
  type = number
}

variable "regions" {
  type = list(string)
}

variable "target_pools" {
  type    = map(map(string))
  default = null
}

variable "machine_type" {
  type    = string
  default = "n1-standard-1"
}

variable "named_ports" {
  type    = map(number)
  default = {}
}

variable "kite_base_image" {
  type    = string
  default = "kite-base-1589586113"
}

variable "guest_accelerator_type" {
  type    = string
  default = null
}

// gcloud compute --project kite-prod-XXXXXXX --format json accelerator-types list | jq '.[] | .zone = (.zone | split("/"))[-1] | {"zone": .zone, "region": .zone[:-2], "name": .name}' | jq -sc '[group_by(.region)[] | {"key": .[0].region, "value": [group_by(.name)[] | {"key": .[0].name, "value": [.[] | .zone]}] | from_entries}] | from_entries'
variable "gpu_zones" {
  type    = map(map(list(string)))
  default = {"asia-east1":{"nvidia-tesla-k80":["asia-east1-a","asia-east1-b"],"nvidia-tesla-p100":["asia-east1-c","asia-east1-a"],"nvidia-tesla-p100-vws":["asia-east1-c","asia-east1-a"],"nvidia-tesla-t4":["asia-east1-c","asia-east1-a"],"nvidia-tesla-t4-vws":["asia-east1-c","asia-east1-a"],"nvidia-tesla-v100":["asia-east1-c"]},"asia-northeast1":{"nvidia-tesla-t4":["asia-northeast1-a","asia-northeast1-c"],"nvidia-tesla-t4-vws":["asia-northeast1-a","asia-northeast1-c"]},"asia-northeast3":{"nvidia-tesla-t4":["asia-northeast3-b","asia-northeast3-c"],"nvidia-tesla-t4-vws":["asia-northeast3-b","asia-northeast3-c"]},"asia-south1":{"nvidia-tesla-t4":["asia-south1-a","asia-south1-b"],"nvidia-tesla-t4-vws":["asia-south1-a","asia-south1-b"]},"asia-southeast1":{"nvidia-tesla-p4":["asia-southeast1-b","asia-southeast1-c"],"nvidia-tesla-p4-vws":["asia-southeast1-b","asia-southeast1-c"],"nvidia-tesla-t4":["asia-southeast1-b","asia-southeast1-c"],"nvidia-tesla-t4-vws":["asia-southeast1-b","asia-southeast1-c"]},"australia-southeast1":{"nvidia-tesla-p100":["australia-southeast1-c"],"nvidia-tesla-p100-vws":["australia-southeast1-c"],"nvidia-tesla-p4":["australia-southeast1-b","australia-southeast1-a"],"nvidia-tesla-p4-vws":["australia-southeast1-b","australia-southeast1-a"],"nvidia-tesla-t4":["australia-southeast1-a"],"nvidia-tesla-t4-vws":["australia-southeast1-a"]},"europe-west1":{"nvidia-tesla-k80":["europe-west1-d","europe-west1-b"],"nvidia-tesla-p100":["europe-west1-d","europe-west1-b"],"nvidia-tesla-p100-vws":["europe-west1-d","europe-west1-b"]},"europe-west2":{"nvidia-tesla-t4":["europe-west2-b","europe-west2-a"],"nvidia-tesla-t4-vws":["europe-west2-b","europe-west2-a"]},"europe-west3":{"nvidia-tesla-t4":["europe-west3-b"],"nvidia-tesla-t4-vws":["europe-west3-b"]},"europe-west4":{"nvidia-tesla-p100":["europe-west4-a"],"nvidia-tesla-p100-vws":["europe-west4-a"],"nvidia-tesla-p4":["europe-west4-c","europe-west4-b"],"nvidia-tesla-p4-vws":["europe-west4-c","europe-west4-b"],"nvidia-tesla-t4":["europe-west4-c","europe-west4-b"],"nvidia-tesla-t4-vws":["europe-west4-c","europe-west4-b"],"nvidia-tesla-v100":["europe-west4-a","europe-west4-c","europe-west4-b"]},"northamerica-northeast1":{"nvidia-tesla-p4":["northamerica-northeast1-c","northamerica-northeast1-b","northamerica-northeast1-a"],"nvidia-tesla-p4-vws":["northamerica-northeast1-c","northamerica-northeast1-b","northamerica-northeast1-a"]},"southamerica-east1":{"nvidia-tesla-t4":["southamerica-east1-c"],"nvidia-tesla-t4-vws":["southamerica-east1-c"]},"us-central1":{"nvidia-tesla-k80":["us-central1-a","us-central1-c"],"nvidia-tesla-p100":["us-central1-c","us-central1-f"],"nvidia-tesla-p100-vws":["us-central1-c","us-central1-f"],"nvidia-tesla-p4":["us-central1-a","us-central1-c"],"nvidia-tesla-p4-vws":["us-central1-a","us-central1-c"],"nvidia-tesla-t4":["us-central1-a","us-central1-b","us-central1-f"],"nvidia-tesla-t4-vws":["us-central1-a","us-central1-b","us-central1-f"],"nvidia-tesla-v100":["us-central1-a","us-central1-c","us-central1-b","us-central1-f"]},"us-east1":{"nvidia-tesla-k80":["us-east1-a","us-east1-c","us-east1-d"],"nvidia-tesla-p100":["us-east1-a","us-east1-b","us-east1-c"],"nvidia-tesla-p100-vws":["us-east1-a","us-east1-b","us-east1-c"],"nvidia-tesla-p4":["us-east1-a"],"nvidia-tesla-p4-vws":["us-east1-a"],"nvidia-tesla-t4":["us-east1-a","us-east1-c","us-east1-d"],"nvidia-tesla-t4-vws":["us-east1-a","us-east1-c","us-east1-d"],"nvidia-tesla-v100":["us-east1-a","us-east1-c"]},"us-east4":{"nvidia-tesla-p4":["us-east4-a","us-east4-c","us-east4-b"],"nvidia-tesla-p4-vws":["us-east4-a","us-east4-c","us-east4-b"],"nvidia-tesla-t4":["us-east4-b"],"nvidia-tesla-t4-vws":["us-east4-b"]},"us-west1":{"nvidia-tesla-k80":["us-west1-b"],"nvidia-tesla-p100":["us-west1-b","us-west1-a"],"nvidia-tesla-p100-vws":["us-west1-b","us-west1-a"],"nvidia-tesla-t4":["us-west1-b","us-west1-a"],"nvidia-tesla-t4-vws":["us-west1-b","us-west1-a"],"nvidia-tesla-v100":["us-west1-b","us-west1-a"]},"us-west2":{"nvidia-tesla-p4":["us-west2-c","us-west2-b"],"nvidia-tesla-p4-vws":["us-west2-c","us-west2-b"]}}
}

variable "boot_disk_size_gb" {
  type = number
  default = 10
}

variable "ubuntu_release" {
  type    = string
  default = "bionic"
}
