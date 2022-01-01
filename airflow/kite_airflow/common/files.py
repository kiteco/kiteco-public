import csv
import codecs

from airflow.hooks.S3_hook import S3Hook

from kite_airflow.common import utils
from kite_airflow.common import configs


def get_scratch_csv_dict_reader(ti, task_id, sub_directory):
    s3 = S3Hook(configs.AWS_CONN_ID)
    filename = ti.xcom_pull(task_ids=task_id)
    s3key = s3.get_key(
        '{}/{}/{}.csv'.format(configs.DIR_SCRATCH_SPACE, sub_directory, filename),
        configs.BUCKET,
    )

    return csv.DictReader(
        codecs.getreader("utf-8")(s3key.get()['Body'])
    )


def get_full_scratch_space_csv(ti, task_id, sub_directory):
    reader = get_scratch_csv_dict_reader(ti, task_id, sub_directory)
    row_list = []

    for row in reader:
        row_list.append(row)

    return row_list


def get_line_of_scratch_space_csv(ti, task_id, sub_directory):
    reader = get_scratch_csv_dict_reader(ti, task_id, sub_directory)
    i = 0

    for row in reader:
        i += 1
        yield i, row


def get_csv_file_as_dict(bucket, file_path):
    s3 = S3Hook(configs.AWS_CONN_ID)
    s3key = s3.get_key(file_path, bucket)
    reader = csv.DictReader(codecs.getreader("utf-8")(s3key.get()['Body']))
    row_list = []

    for row in reader:
        row_list.append(row)

    return row_list


def write_dict_on_csv_file(bucket, file_path, data_list):
    if(len(data_list) == 0):
        return

    s3_hook = S3Hook(configs.AWS_CONN_ID)
    upload_data_list = []

    keys = data_list[0].keys()
    upload_data_list.append(','.join(keys))

    for item in data_list:
       values = item.values()
       upload_data_list.append(','.join(values))

    s3_hook.load_bytes(
        '\n'.join(upload_data_list).encode('utf-8'),
        file_path,
        bucket,
        replace=True,
    )
