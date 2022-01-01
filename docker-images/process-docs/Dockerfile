FROM python:3.4

ADD requirements.txt /
RUN pip install -r requirements.txt
COPY process_docs.py /

CMD ["python", "process_docs.py"]
