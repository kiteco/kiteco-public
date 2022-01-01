FROM python:3.4

ADD github.py /
ADD main.py /
ADD requirements.txt /
RUN pip install -r requirements.txt

ENTRYPOINT python main.py --daily
