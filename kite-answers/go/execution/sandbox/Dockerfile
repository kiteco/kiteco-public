FROM python:3.7.3-stretch

WORKDIR /kite/app

COPY packages.txt .
RUN pip install --no-cache-dir -r packages.txt

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY sample-files ./sample-files

COPY entrypoint.py .

WORKDIR /kite/run
ENTRYPOINT ["python", "/kite/app/entrypoint.py"]
