import datetime
import json
import csv
import codecs

from airflow.hooks.S3_hook import S3Hook

from kite_airflow.common import configs
from kite_airflow.common import utils
from kite_airflow.common import files


BUCKET = 'kite-youtube-data' if utils.is_production() else 'kite-metrics-test'

DIR_PROJECT = 'youtube-dashboard'
DIR_SCRATCH = 'athena-scratch-space'
DIR_CHANNELS = '{}/channels'.format(DIR_PROJECT)
DIR_VIDEOS = '{}/videos'.format(DIR_PROJECT)
DIR_ACTIVITIES = '{}/activities'.format(DIR_PROJECT)
DIR_SNAPSHOTS = '{}/snapshots'.format(DIR_PROJECT)

FILE_CACHED_URLS = '{}/cached_urls.csv'.format(DIR_PROJECT)


def get_scratch_space_csv(ti, task_id):
    '''
    Get file content of a csv in json list
    '''

    s3 = S3Hook(configs.AWS_CONN_ID)
    filename = ti.xcom_pull(task_ids=task_id)
    s3key = s3.get_key(
        '{}/{}.csv'.format(DIR_SCRATCH, filename),
        BUCKET,
    )

    json_list = []
    reader = csv.DictReader(
        codecs.getreader("utf-8")(s3key.get()['Body'])
    )

    for row in reader:
        json_list.append(row)

    return json_list


def write_json_list_on_file(file_path, json_list):
    s3_hook = S3Hook(configs.AWS_CONN_ID)
    data = []

    for index, json_obj in enumerate(json_list):
        data.append(json.dumps(json_obj))

    s3_hook.load_bytes(
        '\n'.join(data).encode('utf-8'),
        file_path,
        BUCKET,
        replace=True,
    )


def get_cached_urls_from_file():
    try:
        cached_urls_list = files.get_csv_file_as_dict(BUCKET, FILE_CACHED_URLS)

    except:
        cached_urls_list = []

    cached_urls_dict = {}
    for cached_url in cached_urls_list:
        cached_urls_dict[cached_url['url']] = bool(cached_url['is_a_kite_redirect'])

    return cached_urls_dict


def write_cached_urls_on_file(cached_urls_dict):
    cached_urls_list = []
    for url, is_kite_redirect in cached_urls_dict.items():
        cached_urls_list.append(
            {
                'url': url,
                'is_a_kite_redirect': 'True' if is_kite_redirect else '' # empty string represents false
            }
        )

    files.write_dict_on_csv_file(BUCKET, FILE_CACHED_URLS, cached_urls_list)


def write_channels_on_file(channel_list):
    write_json_list_on_file(DIR_CHANNELS + '/channels.json', channel_list)


def write_activities_on_file(activity_list):
    file_path = DIR_ACTIVITIES + '/activities' + utils.get_unique_suffix()
    write_json_list_on_file(file_path, activity_list)


def write_videos_on_file(video_list):
    file_path = DIR_VIDEOS + '/videos' + utils.get_unique_suffix()
    write_json_list_on_file(file_path, video_list)


def write_snapshots_on_file(snapshot_list):
    file_path = DIR_SNAPSHOTS + '/snapshots' + utils.get_unique_suffix()
    write_json_list_on_file(file_path, snapshot_list)
