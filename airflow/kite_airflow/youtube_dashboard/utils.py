import re
import requests


def get_video_id_of_search_item(search_item):
    return search_item['id']['videoId']


def get_published_date_of_search_item(search_item):
    return search_item['snippet']['publishedAt']


def get_id_of_video_activity(video_activity):
    return video_activity['contentDetails']['upload']['videoId']


def get_id_of_video_item(video_item):
    return video_item['id']


def get_description_of_video_item(video_item):
    return video_item['snippet']['description']


def get_views_of_video_item(video_item):
    return video_item['statistics'].get('viewCount')


def is_link_present_in_description(video_item, cached_urls_dict):
    '''
    Looks for kite link in the description and in case of shorten url also update the cache
    which we use for performance improvement i.e. prevent future request for same url because
    mostly descriptions of same channel have repetative links

    Returns:\n
        boolean:
            indicates if kite link was present
    '''

    kite_url = 'kite.com'
    description = get_description_of_video_item(video_item)

    # youtubers always uses word Kite in description so if it's not present
    # then no further search is needed
    if 'kite' not in description.lower():
        return False

    if kite_url in description:
        return True

    # some youtubers uses link shortener so for those we uses a combination of cache
    # and HEAD requests to look if kite redirects are present
    urls = re.findall('http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*\(\),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+', description)
    unique_urls = list(dict.fromkeys(urls))

    for url in unique_urls:
        if url in cached_urls_dict:
            if cached_urls_dict[url]:
                return True
            else:
                continue # not returning False because Kite link can be added after we have took the snapshot

        try:
            response = requests.head(url)
            location_header = response.headers.get('Location')
            is_a_kite_redirect = location_header and kite_url in location_header;
            cached_urls_dict[url] = 'True' if is_a_kite_redirect else ''; # empty string represents false

            if is_a_kite_redirect:
                return True

        except Exception:
            cached_urls_dict[url] = '';

    return False
