import time

from kite_airflow.common import utils as common_utils
from kite_airflow.youtube_dashboard import utils


def get_activity_list(yt_client, channel_id, published_after=None, page_token=None):
    '''
    Uses YouTube Activity List API to get activities.

    Returns:\n
        list: activity items
        string: token which we can use to request next page
    '''
    request = yt_client.activities().list(
        part='id,snippet,contentDetails',
        channelId=channel_id,
        maxResults=50,
        publishedAfter=published_after if published_after else common_utils.get_date_time_in_ISO(),
        pageToken=page_token,
    )
    activity_list_response = request.execute()

    return activity_list_response['items'], activity_list_response.get('nextPageToken')


def get_all_activity_list(yt_client, channel_id, published_after=None):
    '''
    Uses YouTube Activity List API to get the list of all activities from given date.

    Returns:\n
        list: all activities found
    '''

    all_activities = []
    next_page_token = None
    exception = None

    try:
        while True:
            activity_list, next_page_token = get_activity_list(
                yt_client,
                channel_id,
                published_after,
                next_page_token,
            )

            if activity_list.count:
                all_activities.extend(activity_list)

            if not next_page_token:
                break

    except Exception as e:
        exception = e

    finally:
        return all_activities, exception


def filter_video_activity_from_list(activity_list):
    '''
    Filters upload video activities from all activities
    '''

    new_upload_video_activity_list = []

    for activity in activity_list:
        if activity['snippet']['type'] == 'upload':
            new_upload_video_activity_list.append(activity)

    return new_upload_video_activity_list


def get_unique_upload_video_activity_list(video_activity_list):
    '''
    Filters duplicated upload video activities.

    Youtube Activity API can send same upload video activity twice
    (don't know the exact reason) and there is no easy way to filter
    them therefore this function is added which filters them based
    on video id's
    '''

    video_ids = set()  # using it to filter videos
    unique_video_activity_list = []

    for video_activity in video_activity_list:
        video_id = utils.get_id_of_video_activity(video_activity)

        if not video_id in video_ids:
            video_ids.add(video_id)
            unique_video_activity_list.append(video_activity)

    return unique_video_activity_list


def get_video_search_list(yt_client, channel_id, published_before=None, page_token=None):
    '''
    Uses YouTube Search List API to get recent videos.

    Returns:\n
        list: searched videos items
        string: token which we can use to request next page
    '''

    request = yt_client.search().list(
        part='snippet',
        channelId=channel_id,
        maxResults=50,
        publishedBefore=published_before if published_before else common_utils.get_date_time_in_ISO(),
        type='video',
        order='date',
        pageToken=page_token,
    )
    video_search_list_response = request.execute()

    return video_search_list_response['items'], video_search_list_response.get('nextPageToken')


def get_all_video_search_list(yt_client, channel_id, published_before, search_budget):
    '''
    Uses YouTube Search List API to get all available videos of a channel

    Returns:\n
        list: all videos of channel
    '''

    no_of_searches = 0
    all_video_searches = []
    next_page_token = None
    has_channel_search_remaining = True
    exception = None

    try:
        while True:
            video_search_list, next_page_token = get_video_search_list(
                yt_client,
                channel_id,
                published_before,
                next_page_token
            )

            has_channel_search_remaining = bool(next_page_token)

            if video_search_list.count:
                all_video_searches.extend(video_search_list)

            if not next_page_token:
                break

            no_of_searches += 1

            if search_budget - no_of_searches <= 0:
                break

    except Exception as e:
        exception = e

    finally:
        return all_video_searches, bool(has_channel_search_remaining), no_of_searches, exception


def get_video_list(yt_client, videos_id_list):
    '''
    Uses YouTube Video List API to get details about the video

    Returns:\n
        list: detailed info of videos
    '''

    request = yt_client.videos().list(
        part='snippet,statistics',
        id=','.join(videos_id_list)
    )
    video_list_response = request.execute()

    return video_list_response['items']
