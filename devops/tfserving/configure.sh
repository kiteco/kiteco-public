#!/bin/bash

# Install headless driver, we don't need Xorg, etc...
sudo add-apt-repository -y ppa:graphics-drivers/ppa
sudo apt-get update
sudo apt-get install -y nvidia-headless-440 nvidia-utils-440
sudo apt-get install -y awscli

# Install nvidia-container-toolkit
sudo apt-get install -y docker.io
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list
sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit
sudo systemctl restart docker
sudo apt-get install unzip

## Add the Cloud SDK distribution URI as a package source
#echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
## Import the Google Cloud Platform public key
#curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key --keyring /usr/share/keyrings/cloud.google.gpg add -
## Update the package list and install the Cloud SDK
#sudo apt-get update && sudo apt-get install -y google-cloud-sdk

wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | sudo apt-key add -
echo "deb https://artifacts.elastic.co/packages/7.x/apt stable main" | sudo tee -a /etc/apt/sources.list.d/elastic-7.x.list
sudo apt-get update && sudo apt-get install metricbeat

## Prometheus export for NVidia GPU monitoring
sudo docker run --gpus all -p 9445:9445 -d -t mindprince/nvidia_gpu_prometheus_exporter:0.1

sudo chown root /home/ubuntu/metricbeat.yml /home/ubuntu/prometheus.yml.disabled /home/ubuntu/prometheus_nvidia.yml.disabled
sudo mv /home/ubuntu/metricbeat.yml /etc/metricbeat/metricbeat.yml
sudo mv /home/ubuntu/*.yml.disabled /etc/metricbeat/modules.d/
sudo metricbeat modules enable prometheus prometheus_nvidia

# We do the secret initialization and metricbeat restart in the metadata script to be sure to have access to AWS for the secret
# aws --region=us-west-1 --query=SecretString --out=text secretsmanager get-secret-value --secret-id beats_elastic_auth_str | sudo metricbeat keystore add cloud_auth --stdin --force
# sudo service metricbeat restart
