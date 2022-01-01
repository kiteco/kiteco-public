import logging
import os
import pandas as pd

from .retention_charts import monthly_retention_plot
from .retention_rate import naive_retention_fn
from .model.pro_model import train_pro_model


def pro_retention_charts(out_dir: str, histories: pd.DataFrame, users: pd.DataFrame):
    model = train_pro_model(users)

    pro_uids = model.get_pro_uids(users)
    pro_users = users[users.index.isin(pro_uids)]
    pro_percent = len(pro_users) / len(users) * 100
    logging.info(f"{pro_percent}% of users classified as professional")

    monthly_retention_plot(
        os.path.join(out_dir, "naive_pro_14d.png"), histories, pro_users, 14, naive_retention_fn,
        "Naive 14-day pro user retention")
