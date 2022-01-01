FROM jupyter/scipy-notebook:ea01ec4d9f57
RUN pip install jupyterlab==2.2.0rc1 && pip install jupyter-kite && jupyter labextension install @kiteco/jupyterlab-kite

USER root
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*
RUN mkdir -p /usr/share/kite && mkdir /usr/share/kite/current
COPY kite-updater.sh /usr/share/kite
RUN chmod a+x /usr/share/kite/kite-updater.sh
RUN /usr/share/kite/kite-updater.sh --noexec --target /usr/share/kite/current && rm /usr/share/kite/kite-updater.sh 

RUN fix-permissions /usr/share/kite
USER jovyan
COPY run_kite.sh /usr/share/kite/run_kite.sh

ENTRYPOINT ["tini", "-g", "--"]
CMD ["/usr/share/kite/run_kite.sh"]
