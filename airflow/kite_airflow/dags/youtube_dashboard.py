import datetime

from airflow import DAG
from airflow.operators.python_operator import PythonOperator
from airflow.contrib.operators.aws_athena_operator import AWSAthenaOperator
import googleapiclient.discovery
from jinja2 import PackageLoader

from kite_airflow.plugins.google import GoogleSheetsRangeOperator
from kite_airflow.common import configs
from kite_airflow.common import utils as common_utils
from kite_airflow.youtube_dashboard import api
from kite_airflow.youtube_dashboard import files
from kite_airflow.youtube_dashboard import utils
from kite_airflow.slack_alerts import task_fail_slack_alert


BUCKET = 'kite-youtube-data' if common_utils.is_production() else 'kite-metrics-test'
SCRATCH_SPACE_LOC = 's3://{}/athena-scratch-space/'.format(BUCKET)

DATABASE = 'prod_kite_link_stats_youtube' if common_utils.is_production() else 'kite_link_stats_youtube'
TABLE_CHANNELS = {
    'name': 'kite_link_stats_youtube_channels',
    'data_location': 's3://{}/youtube-dashboard/channels/'.format(BUCKET),
}
TABLE_VIDEOS = {
    'name': 'kite_link_stats_youtube_videos',
    'data_location': 's3://{}/youtube-dashboard/videos/'.format(BUCKET),
}

default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'start_date': datetime.datetime(2020, 11, 21),
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 0,
    'retry_delay': datetime.timedelta(minutes=5),
    'on_failure_callback': task_fail_slack_alert,
}

kite_link_stats_dag = DAG(
    'youtube_dashboard',
    description='Import links stats of sponsored videos for the YouTube dashboard.',
    default_args=default_args,
    schedule_interval='10 0 * * *',
    jinja_environment_kwargs={
        'loader': PackageLoader('kite_airflow', 'templates')
    },
)

schema_operators = []
for table in [TABLE_CHANNELS, TABLE_VIDEOS]:
    drop_op = AWSAthenaOperator(
        aws_conn_id=configs.AWS_CONN_ID,
        task_id='drop_table_{}'.format(table['name']),
        query='DROP TABLE IF EXISTS {}'.format(table['name']),
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database=DATABASE,
        dag=kite_link_stats_dag,
        params={'data_location': table['data_location']},
    )

    create_op = AWSAthenaOperator(
        aws_conn_id=configs.AWS_CONN_ID,
        task_id='create_table_{}'.format(table['name']),
        query='athena/tables/{}.tmpl.sql'.format(table['name']),
        output_location='s3://kite-metrics-test/athena-results/ddl',
        database=DATABASE,
        dag=kite_link_stats_dag,
        params={'data_location': table['data_location']},
    )

    drop_op >> create_op
    schema_operators.append(create_op)

get_channels_op = AWSAthenaOperator(
    aws_conn_id=configs.AWS_CONN_ID,
    task_id='get_channels',
    query='SELECT * FROM {}'.format(TABLE_CHANNELS['name']),
    output_location=SCRATCH_SPACE_LOC,
    database=DATABASE,
    dag=kite_link_stats_dag,
)
schema_operators >> get_channels_op

get_videos_op = AWSAthenaOperator(
    aws_conn_id=configs.AWS_CONN_ID,
    task_id='get_videos',
    query='SELECT * FROM {}'.format(TABLE_VIDEOS['name']),
    output_location=SCRATCH_SPACE_LOC,
    database=DATABASE,
    dag=kite_link_stats_dag,
)
schema_operators >> get_videos_op

get_channels_sheet_operator = GoogleSheetsRangeOperator(
    gcp_conn_id='google_cloud_kite_dev',
    spreadsheet_id='XXXXXXX-J0',
    range="'List of Channels'!A:C",
    task_id='get_channels_sheet',
    dag=kite_link_stats_dag,
)


def update_videos_from_all_channels(ti, yt_client):
    '''
    Take all given channels and store the list their videos

    In case of new channel we search all videos and in case of an existing
    channel we only search new videos via YouTube activities

    Returns:\n
        list:
            new video items which we will use while taking snapshots. We need this
            because athena queries are evaluating at start so we will not receive
            these new videos via get videos query.
    '''
    channel_list = files.get_scratch_space_csv(ti, get_channels_op.task_id)

    sheet_data = ti.xcom_pull(task_ids='get_channels_sheet')['values']
    cid_field = sheet_data[0].index('Channel ID')
    sheet_channels = {line[cid_field] for line in sheet_data[1:] if len(line) > cid_field and line[cid_field].strip()}

    for new_c in sheet_channels - {c['id'] for c in channel_list}:
        channel_list.append({'id': new_c, 'is_backfilled': 'false', 'last_backfill_until': '', 'last_updated': ''})

    new_video_list = []
    search_budget = 80
    exception = None

    for channel in channel_list:
        channel_id = channel['id']

        # indicates new channel or a channels whose backfilled is yet to complete
        if channel['is_backfilled'] == 'false':

            # incase of backfill was incomplete then resumes where it's left off
            published_before = channel['last_backfill_until'] if channel['is_backfilled'] == 'false' else None

            video_search_list, has_channel_search_remaining, no_of_searches, exception = api.get_all_video_search_list(
                yt_client,
                channel_id,
                published_before,
                search_budget,
            )

            for video_search_item in video_search_list:
                new_video_list.append({
                    'id': utils.get_video_id_of_search_item(video_search_item),
                    'channel_id': channel_id,
                })

            # only update channel attributes if videos are found (also handles YT out of quota cases)
            if(video_search_list):
                last_search_item = video_search_list[- 1]
                channel['last_backfill_until'] = utils.get_published_date_of_search_item(last_search_item)
                channel['is_backfilled'] = not has_channel_search_remaining

                # update the last_updated of channel which will help is in limiting future searches
                channel['last_updated'] = common_utils.get_date_time_in_ISO()

            search_budget -= no_of_searches

            if search_budget <= 0:
                break

        all_activity_list, exception = api.get_all_activity_list(
            yt_client,
            channel_id,
            channel['last_updated'],
        )

        if(len(all_activity_list)):
            files.write_activities_on_file(all_activity_list)

        video_activity_list = api.filter_video_activity_from_list(
            all_activity_list,
        )

        for video_activity in video_activity_list:
            new_video_list.append({
                'id': utils.get_id_of_video_activity(video_activity),
                'channel_id': channel_id,
            })

        # update the last_updated of channel which will help is in limiting future searches
        channel['last_updated'] = common_utils.get_date_time_in_ISO()

    files.write_channels_on_file(channel_list)

    if len(new_video_list) > 0:
        files.write_videos_on_file(new_video_list)

    if exception:
        raise exception

    return new_video_list


def take_snapshots_and_update_files(video_list_for_snapshots, cached_urls_dict):
    snapshot_list = get_snapshots_list(video_list_for_snapshots, cached_urls_dict)
    files.write_snapshots_on_file(snapshot_list)
    files.write_cached_urls_on_file(cached_urls_dict)


def get_snapshots_list(video_list, cached_urls_dict):
    if not video_list:
        return

    snapshot_list = []
    for video_item in video_list:
        snapshot_list.append({
            'video_id': utils.get_id_of_video_item(video_item),
            'description': utils.get_description_of_video_item(video_item),
            'is_link_present': utils.is_link_present_in_description(video_item, cached_urls_dict), # also updates the cache in case of shorten urls
            'views': utils.get_views_of_video_item(video_item),
            'timestamp': common_utils.get_date_time_in_ISO(),
        })

    return snapshot_list


def update_snapshots_of_all_videos(ti, yt_client, new_video_list):
    '''
    Take snapshots of all of the available videos and new videos
    '''

    video_list_for_snapshots = []
    cached_urls_dict = files.get_cached_urls_from_file()
    all_videos_list = files.get_scratch_space_csv(ti, get_videos_op.task_id)
    all_videos_id_list = [video['id'] for video in all_videos_list]
    no_of_batch_requests = 50  # to optimise YouTube quota

    # appending new videos id also because get videos query don't return
    # us new results that are been during the execution of this script
    all_videos_id_list.extend(
        list(map(lambda video: video['id'], new_video_list))
    )

    for start_index in range(0, len(all_videos_id_list), no_of_batch_requests):
        try:
            video_list = []
            end_index = (start_index) + no_of_batch_requests
            videos_id_batch_list = all_videos_id_list[start_index:end_index]
            video_list = api.get_video_list(yt_client, videos_id_batch_list)

            video_list_for_snapshots.extend(video_list)
        except Exception:
            # store data until now in case of any error or if quota exceeded
            take_snapshots_and_update_files(video_list_for_snapshots, cached_urls_dict)
            raise

    take_snapshots_and_update_files(video_list_for_snapshots, cached_urls_dict)


def get_snaphots_of_videos(ti, **context):
    api_service_name = 'youtube'
    api_version = 'v3'
    api_key = 'XXXXXXX'

    yt_client = googleapiclient.discovery.build(
        api_service_name, api_version, developerKey=api_key
    )

    new_video_list = update_videos_from_all_channels(ti, yt_client)
    update_snapshots_of_all_videos(ti, yt_client, new_video_list)


get_snaphots_of_videos_operator = PythonOperator(
    python_callable=get_snaphots_of_videos,
    task_id=get_snaphots_of_videos.__name__,
    dag=kite_link_stats_dag,
    provide_context=True,
)
(
    get_channels_op,
    get_videos_op,
    get_channels_sheet_operator,
) >> get_snaphots_of_videos_operator
