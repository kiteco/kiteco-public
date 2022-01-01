import json
import time

from sources.tools import test_file_exists

import requests


def get_id_from_so_url(url):
    return url.split("/")[4]


def crawl_moz(url_list_path, csv_output_folder):
    list_url = None
    with open("so_url.txt") as flist:
        list_url = json.load(flist)
    if list_url is None:
        print("Impossible to load the list of url, exiting")
        return
    for url in list_url:
        id = get_id_from_so_url(url)
        if test_file_exists(csv_output_folder + id + ".csv"):
            print("The file {} already exists, skipping it".format(id+".csv"))
            continue
        start = time.time()
        get_csv_file(url, id, start, csv_output_folder)


def get_csv_file(url, question_id, start, csv_output_folder):

    print("Processing {}".format(url))
    headers = {
        'origin': 'https://analytics.moz.com',
        'accept-encoding': 'gzip, deflate, br',
        'accept-language': 'en-US,en;q=0.9',
        'user-agent': 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36',
        'content-type': 'application/json;charset=UTF-8',
        'accept': 'application/json, text/plain, */*',
        'referer': 'https://analytics.moz.com/pro/keyword-explorer/site/competitive-keywords?locale=en-US&q=https%3A%2F%2Fstackoverflow.com%2Fquestions%2F3294889%2Fiterating-over-dictionaries-using-for-loops&type=url',
        'authority': 'analytics.moz.com',
        'cookie': 'XXXXXXX',
    }

    data = '{{"filters":{{}},"sort":{{"by":"primary_rank","reverse":false}},"subjects":{{"primary":{{"locale":"en-US","scope":"url","target":"{}"}},"secondaries":[]}}}}'.format(url)
    print(data)
    response = requests.post('https://analytics.moz.com/pro/keyword-explorer/api/2.5/site/rankings.csv', headers=headers, data=data)
    response.raise_for_status()
    with open(csv_output_folder + question_id + ".csv", "w") as outfile:
        outfile.write(response.text)
        print("CSV saved to file {}".format(csv_output_folder + question_id + ".csv"))
    end = time.time()
    print("Time to process query {:.3f}s".format(end-start))

def get_moz_data_for_url_list(url_list_path, output_folder):
    crawl_moz(url_list_path, output_folder)


