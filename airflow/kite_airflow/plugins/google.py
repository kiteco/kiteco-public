# from airflow.models.baseoperator import BaseOperator
# from airflow.utils.decorators import apply_defaults

import time

from googleapiclient.discovery import build

from airflow import AirflowException
from airflow.contrib.hooks.gcp_api_base_hook import GoogleCloudBaseHook
from airflow.plugins_manager import AirflowPlugin
from airflow.models.baseoperator import BaseOperator
from airflow.utils.decorators import apply_defaults


class GoogleSheetsHook(GoogleCloudBaseHook):
    _conn = None

    def __init__(self, api_version="v4", gcp_conn_id="google_cloud_default", delegate_to=None):
        super(GoogleSheetsHook, self).__init__(gcp_conn_id, delegate_to)
        self.api_version = api_version

    def get_conn(self):
        """
        Retrieves the connection to Cloud Functions.

        :return: Google Cloud Build services object.
        """
        if not self._conn:
            http_authorized = self._authorize()
            self._conn = build('sheets', self.api_version, http=http_authorized, cache_discovery=False)
        return self._conn

    @GoogleCloudBaseHook.fallback_to_default_project_id
    def get_range(self, spreadsheet_id:str, range:str, **kwargs):
        conn = self.get_conn()

        sheets = conn.spreadsheets().values()
        return sheets.get(spreadsheetId=spreadsheet_id, range=range).execute(num_retries=self.num_retries)


class GoogleSheetsRangeOperator(BaseOperator):

    @apply_defaults
    def __init__(
            self,
            spreadsheet_id: str,
            range: str,
            gcp_conn_id: str = 'google_cloud_default',
            *args, **kwargs) -> None:
        super().__init__(*args, **kwargs)
        self.gcp_conn_id = gcp_conn_id
        self.spreadsheet_id = spreadsheet_id
        self.range=range

    def execute(self, context):
        hook = GoogleSheetsHook(gcp_conn_id=self.gcp_conn_id)
        return hook.get_range(spreadsheet_id=self.spreadsheet_id, range=self.range)


class GoogleSheetsPlugin(AirflowPlugin):
    name = 'google_sheets'
    operators = [GoogleSheetsRangeOperator]
    hooks = [GoogleSheetsHook]