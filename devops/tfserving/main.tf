terraform {
  backend "s3" {
    bucket = "kite-terraform-state"
    key    = "tf_serving/terraform.tfstate"
    region = "us-west-1"
  }
}

provider "google" {
  project = "kite-dev-XXXXXXX"
  region  = "us-west1"
  zone    = "us-west1-b"
}

provider "google-beta" {
  region  = "us-west1"
  project = "kite-dev-XXXXXXX"
}

// Terraform plugin for creating random ids
resource "random_id" "instance_id" {
  byte_length = 8
}

data "google_compute_address" "tfserving_static_ip" {
  name = "tfserving-dev"
}

module "instance_role" {
  source  = "../terraform/cloud/modules/instance_role"
  name    = "tfserving"
  secrets = []

  policy_statements = [{
    sid = "100"
    actions = ["s3:GetObject"]
    resources = ["arn:aws:s3:::kite-data/*"]
  }, {
    sid = "101"
    actions = ["s3:ListBucket"]
    resources = ["arn:aws:s3:::kite-data"]
  }]
}

resource "google_compute_instance" "tf_serving_instance" {
  name         = "tfserving-dev"
  machine_type = "n1-standard-4"
  zone         = "us-west1-b"

  tags = ["tfserving"]

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2004-lts"
    }
  }

  network_interface {
    network = "kite-dev"
    subnetwork = "kite-dev-private-us-west1"

    access_config {
      nat_ip = data.google_compute_address.tfserving_static_ip.address
    }
  }

  service_account {
    scopes = ["cloud-platform"]
    email = module.instance_role.service_account_email
  }

  scheduling {
    on_host_maintenance = "TERMINATE"
  }

  guest_accelerator {
    count = 1
    type = "nvidia-tesla-t4"
  }


  provisioner "remote-exec" {
    inline = [
      "cd ~",
      "mkdir tfserving"
    ]

    connection {
      type        = "ssh"
      host        = self.network_interface[0].network_ip
      user        = "ubuntu"
      private_key = file(var.private_key_path)
    }
  }

  provisioner "file" {
    source      = "configure.sh"
    destination = "/home/ubuntu/configure.sh"

    connection {
      type        = "ssh"
      host        = self.network_interface[0].network_ip
      user        = "ubuntu"
      private_key = file(var.private_key_path)
    }
  }

  provisioner "file" {
    source      = var.model_config_list_path
    destination = "/home/ubuntu/tfserving/model_config_list.txt"

    connection {
      type        = "ssh"
      host        = self.network_interface[0].network_ip
      user        = "ubuntu"
      private_key = file(var.private_key_path)
    }
  }

  provisioner "file" {
    source      = "monitoring_config.txt"
    destination = "/home/ubuntu/tfserving/monitoring_config.txt"

    connection {
      type        = "ssh"
      host        = self.network_interface[0].network_ip
      user        = "ubuntu"
      private_key = file(var.private_key_path)
    }
  }

  provisioner "file" {
    source      = "run.sh"
    destination = "/home/ubuntu/tfserving/run.sh"

    connection {
      type        = "ssh"
      host        = self.network_interface[0].network_ip
      user        = "ubuntu"
      private_key = file(var.private_key_path)
    }
  }

  provisioner "file" {
    source      = "metricbeat.yml"
    destination = "/home/ubuntu/metricbeat.yml"

    connection {
      type        = "ssh"
      host        = self.network_interface[0].network_ip
      user        = "ubuntu"
      private_key = file(var.private_key_path)
    }
  }

  provisioner "file" {
    source      = "prometheus.yml.disabled"
    destination = "/home/ubuntu/prometheus.yml.disabled"

    connection {
      type        = "ssh"
      host        = self.network_interface[0].network_ip
      user        = "ubuntu"
      private_key = file(var.private_key_path)
    }
  }

  provisioner "file" {
    source      = "prometheus_nvidia.yml.disabled"
    destination = "/home/ubuntu/prometheus_nvidia.yml.disabled"

    connection {
      type        = "ssh"
      host        = self.network_interface[0].network_ip
      user        = "ubuntu"
      private_key = file(var.private_key_path)
    }
  }

  provisioner "remote-exec" {
    inline = [
    "while [ ! -f /home/ubuntu/PROVISIONING_DONE ]",
      "do",
      "echo \"Checking for metadata_startup_script to complete, sleeping 30s\"",
      "sleep 30;",
     "done"
    ]

    connection {
      type        = "ssh"
      host        = self.network_interface[0].network_ip
      user        = "ubuntu"
      private_key = file(var.private_key_path)
    }
  }

  metadata_startup_script = templatefile("templates/download_run.tmpl",
  { aws_acces_key_id : module.instance_role.aws_acces_key_id, gcp_aws_secret_access_key : module.instance_role.gcp_aws_secret_access_key})

}
