from airflow.contrib.operators.slack_webhook_operator import SlackWebhookOperator
from airflow.hooks.base_hook import BaseHook
from airflow.models import Variable


SLACK_CONN_ID = "slack_devops_notifications"


def task_fail_slack_alert(context):
    """
    Callback task that can be used in DAG to alert of failure task completion
    Args:
        context (dict): Context variable passed in from Airflow
    Returns:
        None: Calls the SlackWebhookOperator execute method internally
    """

    if Variable.get('env', 'dev') == 'dev':
        return

    slack_webhook_token = BaseHook.get_connection(SLACK_CONN_ID).password
    slack_msg = """
            :red_circle: Task Failed.
            *Task*: {task}
            *Dag*: {dag} (https://airflow.kite.dev/admin/airflow/tree?dag_id={dag})
            *Execution Time*: {exec_date}
            """.format(
        task=context.get("task_instance").task_id,
        dag=context.get("task_instance").dag_id,
        exec_date=context.get("execution_date"),
    )

    failed_alert = SlackWebhookOperator(
        task_id="slack_test",
        http_conn_id=SLACK_CONN_ID,
        webhook_token=slack_webhook_token,
        message=slack_msg,
        username="airflow",
    )

    return failed_alert.execute(context=context)
