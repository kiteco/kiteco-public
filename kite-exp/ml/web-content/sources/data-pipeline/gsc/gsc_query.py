from googleapiclient import discovery

import apiclient

from oauth2client import client
from oauth2client import file
from oauth2client import tools

import json
import sys
import httplib2


backoff_timing = [1,2,3,5,8,13,21,34,55, 600, -1]
START_DATE = "2019-03-01"
END_DATE = "2019-04-01"

KITE_ADDR = "https://kite.com/"

CLIENT_SECRET_PATH = "/home/moe/workspace/web-content/source/client_secret.json"
APP_DATA_PATH = "/home/moe/workspace/web-content/source/web-content.dat"


def list_pages(start_date, end_date, output_file, service):
    i = 0
    pages = []
    try:
        while True:

            request = {
                'startDate': start_date,
                'endDate': end_date,
                'dimensions': ['page'],
                'rowLimit': 25000,
                'startRow': i*25000,
            }
            i += 1
            response = execute_request(service, KITE_ADDR, request)
            if response is not None and 'rows' in response and len(response['rows']) > 0:
                pages.extend(response["rows"])
            else:
                break
    except:
        e = sys.exc_info()[0]
        print("Error while fetching pages info: {}".format(e))

    with open(output_file, 'w') as outfile:
        json.dump(pages, outfile)

    print("{} pages received, list written to {}".format(len(pages), output_file))


def list_queries(target_page, start_date, end_date, service, output_file):
    i = 0
    queries = []
    try:
        while True:

            request = {
                'startDate': start_date,
                'endDate': end_date,
                'dimensions': ['query'],
                'rowLimit': 25000,
                'startRow': i*25000,
                'dimensionFilterGroups': [
                    {
                        "filters": [
                            {
                                "dimension": "page",
                                "expression": target_page,
                                "operator": "equals"
                            }
                        ]
                    }
                ]
            }
            i += 1

            response = execute_request(service, KITE_ADDR, request)
            if response is not None and 'rows' in response and len(response['rows']) > 0:
                queries.extend(response["rows"])
                if len(response["rows"]) < 25000:
                    break
            else:
                break
    except Exception as e:
        print("Error while fetching pages info: {}".format(e))

    if output_file:
        with open(output_file, 'w') as outfile:
            json.dump(queries, outfile)

    print("{} queries for the page {}".format(len(queries), target_page))
    return queries


def get_most_frequent_queries(service, start_date, end_date, outpath, only_usa = False, only_examples=True,
                              dimensions: list=['query', 'page']):
    """
    Extract all the most frequent queries leading to Kite website
    For the time interval between start_date and end_date
    :param service: Google service object, get it by calling init_google_service function
    :param start_date: Begin of the time interval to consider
    :param end_date: End of the time interval to consider
    :param outpath: File where to write the results
    :param only_usa: Should it be only query from the USA (default False)
    :param dimensions: What dimensions to consider. Using ['query', 'page'] is recommended as 'query' doesn't return all queries
    :return:
    """
    queries = []
    i = 0
    while True:
        request = {
                'startDate': start_date,
                'endDate': end_date,
                'dimensions': dimensions,
                'rowLimit': 25000,
                'startRow': i*25000,
                'dimensionFilterGroups': [
                    {
                        "filters": []
                    }
                ]
            }
        if only_usa:
            request["dimensionFilterGroups"][0]["filters"].append({
                "dimension": "country",
                "expression": "USA",
                "operator": "equals"
            })
        if only_examples:
            request["dimensionFilterGroups"][0]["filters"].append(
                {
                    "dimension": "page",
                    "expression": "/examples/",
                    "operator": "contains"
                }
            )
        response = execute_request(service, KITE_ADDR, request)
        if response is not None and 'rows' in response and len(response['rows']) > 0:
            queries.extend(response["rows"])
            if len(response["rows"]) < 25000:
                print("Last response contained only {}, breaking".format(len(response["rows"])))
                break
        else:
            break
        i += 1

    print("Received {} rows, writing them in {}".format(len(queries), outpath))
    with open(outpath, "w") as outfile:
        json.dump(queries, outfile)


def extract_all_queries(start_date, end_date, page_list, service, outfile):
    result = {}
    try:
        with open(page_list, "r") as infile:
            next_page = infile.readline().strip()
            while next_page:
                result[next_page] = list_queries(next_page, start_date, end_date, service, None)
                next_page = infile.readline().strip()
    except Exception as e:
        print("Error while fetching queries: {}".format(e))

    with open(outfile, "w") as outfile:
        json.dump(result, outfile)


def execute_request(service, property_uri, request):
    """Executes a searchAnalytics.query request.
    Args:
      service: The webmasters service to use when executing the query.
      property_uri: The site or app URI to request data for.
      request: The request to be executed.
    Returns:
      An array of response rows.
    """
    return build_searchanalytics_query(service, property_uri, request).execute()


def build_searchanalytics_query(service, property_uri, request):
    """Build a searchAnalytics.query request.
    Args:
      service: The webmasters service to use when executing the query.
      property_uri: The site or app URI to request data for.
      request: The request to be executed.
    Returns:
      An array of response rows.
    """
    return service.searchanalytics().query(
        siteUrl=property_uri, body=request)


def init_google_service():

    scope='https://www.googleapis.com/auth/webmasters'
    # Parser command-line arguments.

    # Name of a file containing the OAuth 2.0 information for this
    # application, including client_id and client_secret, which are found
    # on the API Access tab on the Google APIs
    # Console <http://code.google.com/apis/console>.

    # Set up a Flow object to be used if we need to authenticate.
    flow = client.flow_from_clientsecrets(CLIENT_SECRET_PATH,
                                          scope=scope,
                                          message="Can't find client secrets file")

    # Prepare credentials, and authorize HTTP object with them.
    # If the credentials don't exist or are invalid run through the native client
    # flow. The Storage object will ensure that if successful the good
    # credentials will get written back to a file.
    storage = file.Storage(APP_DATA_PATH)
    credentials = storage.get()
    if credentials is None or credentials.invalid:
        credentials = tools.run_flow(flow, storage)

    def build_request(http, *args, **kwargs):
        storage = file.Storage(APP_DATA_PATH)
        credentials = storage.get()
        http = credentials.authorize(http=httplib2.Http())
        return apiclient.http.HttpRequest(http, *args, **kwargs)
    # Construct a service object via the discovery service.
    service = discovery.build("webmasters", "v3", requestBuilder=build_request, credentials=credentials)
    return service
