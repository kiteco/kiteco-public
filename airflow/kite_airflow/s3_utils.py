import gzip
import json
from airflow.hooks.S3_hook import S3Hook
from airflow.contrib.operators.s3_delete_objects_operator import S3DeleteObjectsOperator


def read_s3_json_files(bucket, file_list):
    s3 = S3Hook('aws_us_east_1')
    
    for file in sorted(file_list):
        obj = s3.get_key(file, bucket)
        for line in gzip.open(obj.get()['Body']):
            rec = json.loads(line)
            to_clean = [rec]
            while to_clean:
                this = to_clean.pop()
                for k in list(this.keys()):
                    v = this[k]
                    if isinstance(v, dict):
                        to_clean.append(v)
                        continue
                    if v is None:
                        del this[k]
            yield rec


class S3DeletePrefixOperator(S3DeleteObjectsOperator):
    def execute(self, context):
        if isinstance(self.keys, str):
            hook = S3Hook(aws_conn_id=self.aws_conn_id, verify=self.verify)
            self.keys = hook.list_keys(bucket_name=self.bucket, prefix=self.keys)
        return super(S3DeletePrefixOperator, self).execute(context)
