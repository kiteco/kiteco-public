FROM python:3.4

ADD github.py /
ADD assign_prs.py /
ADD requirements.txt /
RUN pip install -r requirements.txt

ENTRYPOINT python assign_prs.py
