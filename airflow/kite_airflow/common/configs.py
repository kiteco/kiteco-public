from airflow.models import Variable
from kite_airflow.common import utils


CIO_CREDENTIALS = Variable.get('cio_credentials' if utils.is_production() else 'cio_credentials_dev', deserialize_json=True)
CIO_MAX_CONCURRENT_REQUESTS = 50

MP_CREDENTIALS = Variable.get('mixpanel_credentials' if utils.is_production() else 'mixpanel_credentials_dev', deserialize_json=True)
MP_MAX_CONCURRENT_REQUESTS = 100

# S3
AWS_CONN_ID = 'aws_us_east_1'
BUCKET = 'kite-metrics' if utils.is_production() else 'kite-metrics-test'
DIR_SCRATCH_SPACE = 'athena-scratch-space'
DIR_SCRATCH_URI = 's3://{}/{}'.format(BUCKET, DIR_SCRATCH_SPACE)

# Athena
DB_KITE_METRICS = 'kite_metrics'
