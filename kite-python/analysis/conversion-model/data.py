import time

import boto3


def main():
    with open("query.sql", "r") as fp:
        query = fp.read()

    db = KiteMetricsDB()
    query_id = db.pose_query(query)
    status = db.get_query_status(query_id)
    assert status == "SUCCEEDED", f"Query status: {status}"
    results = db.get_results(query_id)

    with open("train.csv", "wb") as fp:
        fp.write(results)


class KiteMetricsDB:
    def __init__(self):
        self.database = "kite_metrics"
        self.bucket = "kite-metrics-test"
        self.prefix = "athena/conversion-model"
        self.athena = boto3.client("athena", region_name="us-east-1")
        self.s3 = boto3.client("s3")

    def pose_query(self, query):
        location = f"s3://{self.bucket}/{self.prefix}/"
        resp = self.athena.start_query_execution(
            QueryString=query,
            QueryExecutionContext={"Database": self.database},
            ResultConfiguration={"OutputLocation": location},
        )
        return resp["QueryExecutionId"]

    def get_query_status(self, query_id, retries=10):
        time.sleep(60)
        resp = self.athena.get_query_execution(QueryExecutionId=query_id)
        status = resp["QueryExecution"]["Status"]["State"]
        if status in ("QUEUED", "RUNNING") and retries > 0:
            return self.get_query_status(query_id, retries-1)
        return status

    def get_results(self, query_id):
        resp = self.s3.get_object(
            Bucket=self.bucket,
            Key=f"{self.prefix}/{query_id}.csv",
        )
        return resp["Body"].read()


if __name__ == "__main__":
    main()
