#!/usr/bin/env python3

KITE_TELEMETRY_ES = {
    'host': "https://search-kite-telemetry-dev-3-"
            "XXXXXXX.us-east-1.es.amazonaws.com",
    'region': 'us-east-1',
}


def build_client():
    import os
    import boto3
    from elasticsearch import Elasticsearch, RequestsHttpConnection
    from requests_aws4auth import AWS4Auth

    session = boto3.Session()
    credentials = session.get_credentials()

    awsauth = AWS4Auth(credentials.access_key,
                       credentials.secret_key,
                       KITE_TELEMETRY_ES['region'],
                       'es')

    es_client = Elasticsearch(
        hosts=[KITE_TELEMETRY_ES['host']],
        http_auth=awsauth,
        use_ssl=True,
        verify_certs=True,
        connection_class=RequestsHttpConnection,
        retry_on_timeout=True,
        timeout=60,
    )
    return es_client


def make_query(after=None):
    base = {
      "query": {
        "range": {
          "timestamp": {
            "gte": "2020-01-22"
          }
        }
      },
      "size": 0,
      "aggs": {
        "users": {
          "composite": {
            "sources": [
              {
                "user_id": {
                  "terms": {
                    "field": "properties.user_id"
                  }
                }
              }
            ],
            "size": 100
          },
          "aggs": {
            "ips": {
              "terms": {
                "field": "context.ip",
                "size": 10
              }
            }
          }
        }
      }
    }
    if after is not None:
        base["aggs"]["users"]["composite"]["after"] = after
    return base


def main():
    import json

    cli = build_client()

    users = []
    after = None
    while True:
        print(len(users), after)
        resp = cli.search(body=make_query(after=after),
                          index="kite_status_active_*")
        print(resp['_shards'])
        if resp['_shards']['failed']:
            break
        if not resp['aggregations']['users'].get('buckets'):
            break
        for user in resp['aggregations']['users']['buckets']:
            if len(user['ips']['buckets']):
                users.append(user)
        if not resp['aggregations']['users'].get('after_key'):
            break
        after = resp['aggregations']['users']['after_key']

    with open('IPs.json', 'w') as f:
        json.dump(users, f)


if __name__ == '__main__':
    main()
