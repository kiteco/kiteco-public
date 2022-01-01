FROM tensorflow/serving:1.15.0-gpu as tfserving
FROM tensorflow/tensorflow:1.15.2-gpu-py3

# Setup python environment
RUN pip install --upgrade pip
RUN pip install scipy awscli tensorflow-serving-api-gpu==1.15.0

# Install tensorflow c libraries
RUN curl -L "https://storage.googleapis.com/tensorflow/libtensorflow/libtensorflow-gpu-linux-x86_64-1.15.0.tar.gz" | tar -C /usr/local -xz

# Install kite-ml python libs
WORKDIR /
ADD kite-python/kite_ml /kite_ml
RUN cd /kite_ml && pip install -e .

# Copy over tfserving binary and entrypoint script
COPY --from=tfserving /usr/bin/tensorflow_model_server /usr/bin/tensorflow_model_server

# Copy models
COPY kite-server/tunable-models/models/all-langs-large /models/all-langs-large
COPY kite-server/tunable-models/models/go-large /models/go-large
COPY kite-server/tunable-models/models/py-large /models/py-large
COPY kite-server/tunable-models/models/js-large /models/js-large

# These config file paths line up with hard coded paths in teams-server for launching tfserving
COPY kite-server/tunable-models/model.config /tfserving-config/model.config
COPY kite-server/tunable-models/monitoring.config /tfserving-config/monitoring.config
COPY kite-server/tunable-models/batching_parameters.config /tfserving-config/batching_parameters.config

# Install finetuning pipeline
WORKDIR /
ADD kite-server/tunable-models/localtrain /kite_bin/localtrain
ADD kite-server/tunable-models/kite-team-server /kite_bin/kite-team-server

ADD local-pipelines/lexical/train/Makefile.tuned /train/
ADD local-pipelines/lexical/train/model /train/model
ADD local-pipelines/lexical/train/*.py /train/

WORKDIR /train

ENV PATH="/kite_bin:${PATH}"
ENV LD_LIBRARY_PATH="/usr/local/lib:${LD_LIBRARY_PATH}"
ENV CUDA_CACHE_PATH=/cudacache
ENV CUDA_CACHE_MAXSIZE=2147483647

RUN mkdir -p /tuned-models /repositories

ENTRYPOINT ["kite-team-server"]
