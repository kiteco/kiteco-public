#!/bin/bash
docker run -t --rm \
    --gpus all \
    -d \
    -v $PWD:/workdir \
    -p 8501:8501 \
    -p 8500:8500 \
    -e AWS_LOG_LEVEL=3 \
    -e AWS_REGION=us-west-1 \
    --mount type=bind,source=$PWD/model_config_list.txt,target=/models/model_config_list.txt \
    --mount type=bind,source=$PWD/monitoring_config.txt,target=/models/monitoring_config.txt \
    --mount type=bind,source=$PWD/../.aws/credentials,target=/root/.aws/credentials \
    -t tensorflow/serving:latest-gpu \
    --tensorflow_intra_op_parallelism=16 \
    --tensorflow_inter_op_parallelism=16 \
    --model_config_file=/models/model_config_list.txt \
    --monitoring_config_file=/models/monitoring_config.txt \
