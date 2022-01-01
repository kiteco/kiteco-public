FROM apache/airflow:1.10.12

USER airflow

ARG KITECO=${AIRFLOW_HOME}/kiteco
WORKDIR ${KITECO}

COPY --chown=airflow:root airflow/requirements.txt airflow/MANIFEST.in airflow/setup.py ./airflow/
COPY --chown=airflow:root airflow/kite_airflow ./airflow/kite_airflow
COPY --chown=airflow:root kite-python/metrics ./kite-python/metrics

RUN python -m pip install --user --upgrade pip && \
    python -m pip install --user --no-cache-dir -r airflow/requirements.txt && \
    python -m pip install --user ./airflow/ && \
    python -m pip install --user ./kite-python/metrics/

WORKDIR ${AIRFLOW_HOME}

RUN rm -rf dags
RUN ln -s ${KITECO}/airflow/kite_airflow/dags dags

RUN mkdir conf
COPY airflow/conf/prod/airflow.cfg conf/
VOLUME /opt/airflow/conf
RUN ln -s conf/airflow.cfg .
