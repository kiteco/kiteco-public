import numpy as np
import pandas as pd

import plotly
from plotly import offline

import math

import plotly.graph_objs as go
from plotly.offline import download_plotlyjs, init_notebook_mode, plot

import os


AVERAGE_KITE_CTR = 0.04597767391467217
# Obtained by doing total_clicks / total_imps over the 5k top queries bringing the most clicks to kite examples
# Data from March 2019

def extract_qid(s):
    if not s:
        return np.NaN
    s = str(s)
    parts = s.split('/')
    if len(parts) < 5:
        return np.NaN
    if not parts[4].isdigit() or parts[3] != "questions" or parts[2] != "stackoverflow.com":
        return np.NaN
    return int(parts[4])

def get_df_data(df):
    result = []
    for c in df:
        result.append(df[c].tolist())
    return result



class WebViewerGenerator(object):

    def __init__(self, pages_with_topic, topics_model, predicted_imps, so_top5k, output_folder="output", topics_file="webviewer_topics.html", so_posts_file="webviewer_so_posts.html", max_items_per_page=500):
        self.max_items_per_page = max_items_per_page
        self.so_posts_file = so_posts_file
        self.output_folder = output_folder
        self.topics_file = topics_file
        self.so_top5k = so_top5k
        self.pages_with_topic = pages_with_topic
        self.topics_model = topics_model
        self.predicted_click_so_posts = predicted_imps
        self.topics = self.prepare_dataset()

    def get_topic_terms(self, topic_id):
        terms = self.topics_model.get_topic_terms(topic_id)
        result = []
        for i in range(10):
            if terms[i][1] < 0.01:
                break
            result.append("{} ({:.3f})".format(self.topics_model.id2word[terms[i][0]], round(terms[i][1], 4)))
        return "<br>".join(result)

    def prepare_dataset(self):
        qid = self.pages_with_topic[self.pages_with_topic.page_type == 'so'].URL.apply(extract_qid)
        so_page_to_remove = qid[qid.apply(lambda id: len(self.so_top5k[self.so_top5k.PostID == id]) == 0)].index
        self.pages_with_topic.drop(so_page_to_remove, inplace = True)
        clicks_per_posts = qid.apply(lambda i: self.so_top5k.loc[self.so_top5k.PostID == i, 'newViews'].iloc[0]
                                                if (self.so_top5k.PostID == i).sum() > 0 else 0)
        self.pages_with_topic.loc[self.pages_with_topic.page_type == 'so', 'clicks'] = clicks_per_posts
        self.predicted_click_so_posts = self.predicted_click_so_posts.reset_index()
        self.predicted_click_so_posts.columns = ["URL", "kite_predicted_impressions"]
        self.pages_with_topic = self.pages_with_topic.merge(self.predicted_click_so_posts, 'left', on="URL")
        self.pages_with_topic["kite_predicted_clicks"] = self.pages_with_topic.kite_predicted_impressions * AVERAGE_KITE_CTR
        topics_groups = self.pages_with_topic.groupby(self.pages_with_topic.dominant_topic)

        def format_url_list(url_list, volume_list = None):
            result = []
            for i, u in enumerate(url_list):
                volume = ""
                if volume_list:
                    volume = " ({})".format(volume_list[i])
                result.append("<a href=\"{}\">{}{}</a>".format(u, u.split('/')[-1], volume))
            return "<br>".join(result)

        def get_topic_features(t):
            topic_id = t.dominant_topic.max()
            kite_pages = t[t.page_type == 'kite_examples']
            so_pages = t[t.page_type == 'so'].sort_values("clicks", ascending=False)
            kite_volume = kite_pages.volume.sum()
            so_volume = so_pages.volume.sum()
            kite_clicks = kite_pages.clicks.sum()
            so_clicks = so_pages.clicks.sum()
            kite_predicted_clicks = so_pages.kite_predicted_clicks.sum()
            # kite_predicted_impressions = so_pages.kite_predicted_impressions.sum()
            # clicks_ratio = kite_clicks / so_clicks if so_clicks > 0 else 1
            # volume_ratio = kite_volume / so_volume if so_volume > 0 else 1
            so_rank = (so_pages['rank']*so_pages.volume).sum() / so_volume if so_volume > 0 else 0
            return pd.Series(dict(
                # kite_volume=kite_volume,
                # kite_predicted_volume=kite_predicted_impressions,
                # so_volume=so_volume,
                kite_clicks=kite_clicks,
                kite_predicted_clicks=int(kite_predicted_clicks),
                so_clicks=so_clicks,
                # clicks_ratio=clicks_ratio,
                # volume_ratio=volume_ratio,
                so_rank=round(so_rank, 4),
                kite_urls=format_url_list(kite_pages.URL.tolist()),
                so_urls=format_url_list(so_pages.URL.tolist(), so_pages.clicks.tolist()),
                topic_id="{}".format(topic_id),
                topic_terms=self.get_topic_terms(topic_id)
            ))

        topics = topics_groups.apply(get_topic_features)
        # topics["available_clicks"] = topics.kite_predicted_clicks - topics.kite_clicks
        return topics

    def generate_topics_viewer(self):
        columns_and_size = [# ('Kite vol', 60),
                            # ('Kite pred vol', 60),
                            # ('SO volume', 60),
                            ('Kite clicks', 60),
                            ('Kite pred clicks', 80),
                            ('SO Views', 80),
                            # ('Click ratio', 80),
                            # ('Vol ratio', 80),
                            ('SO rank', 60),
                            ('kite URLs', 600),
                            ('SO URLs', 600),
                            ('Topic ID', 140),
                            ('Topic terms', 100),
                            # ('Available Clicks', 70)
                            ]

        columns, sizes = zip(*columns_and_size)
        data_cols = get_df_data(self.topics.sort_values("so_clicks", ascending=False))

        self.topics.sort_values("so_clicks", ascending=False).to_csv(os.path.join(self.output_folder, "topics.csv"), index=False)
        if len(data_cols[0]) <= self.max_items_per_page:
            trace = go.Table(
                columnwidth=sizes,
                header=dict(values=list(columns),
                            fill = dict(color='#C2D4FF')),
                cells=dict(values=data_cols,
                           fill = dict(color='#F5F8FF'),
                           align = ['left'] * 5))

            data = [trace]
            plot(data, filename=os.path.join(self.output_folder, self.topics_file))
        else:
            # Sharding mode
            data_cols = np.array(data_cols)
            for i in range(math.ceil(len(data_cols[0])/self.max_items_per_page)):
                name = "{}_{}.html".format(self.topics_file[:-5], i+1)
                trace = go.Table(
                    columnwidth=sizes,
                    header=dict(values=list(columns),
                                fill = dict(color='#C2D4FF')),
                    cells=dict(values=data_cols[:, self.max_items_per_page*i:self.max_items_per_page*(i+1)],
                               fill = dict(color='#F5F8FF'),
                               align = ['left'] * 5))

                data = [trace]
                plot(data, filename=os.path.join(self.output_folder, name))

    def generate_so_posts_viewer(self):
        df_pages = self.prepare_so_pages_dataset()
        df_pages.to_csv(os.path.join(self.output_folder, "so_posts.csv"), index=False)
        columns_and_size = [
                             ('SO Views', 80),
                             # ('SO volume', 60),
                             # ('Kite pred vol', 60),
                             ('Kite pred clicks', 80),
                             ('SO rank', 60),
                             ('SO URL', 600),
                             ('Dominant Topic', 140),
                             ('Topic terms', 100)]
        columns, sizes = zip(*columns_and_size)
        data_cols = get_df_data(df_pages)
        hover_text = df_pages.topic_terms.tolist()
        print("data cols lenght: {} and max items: {}".format(len(data_cols[0]), self.max_items_per_page))
        if len(data_cols[0]) <= self.max_items_per_page:
            trace = go.Table(
                columnwidth=sizes,
                header=dict(values=list(columns),
                            fill = dict(color='#C2D4FF')),
                cells=dict(values=data_cols,
                           fill = dict(color='#F5F8FF'),
                           align = ['left'] * 5))
            data = [trace]
            plot(data, filename=os.path.join(self.output_folder, self.so_posts_file))
        else:
            # Sharding
            data_cols = np.array(data_cols)
            for i in range(math.ceil(len(data_cols[0])/self.max_items_per_page)):
                name = "{}_{}.html".format(self.so_posts_file[:-5], i+1)
                chunk = data_cols[:, self.max_items_per_page*i:self.max_items_per_page*(i+1)]
                trace = go.Table(
                    columnwidth=sizes,
                    header=dict(values=list(columns),
                                fill = dict(color='#C2D4FF')),
                    cells=dict(values=chunk,
                               fill=dict(color='#F5F8FF'),
                               align=['left'] * 5))
                data = [trace]
                plot(data, filename=os.path.join(self.output_folder, name))


    def prepare_so_pages_dataset(self):
        pages = self.pages_with_topic[self.pages_with_topic.page_type == 'so'].copy()
        pages.kite_predicted_clicks = pages.kite_predicted_clicks.astype(np.int64)
        pages = pages[["clicks", "kite_predicted_clicks", "rank", "URL", "dominant_topic"]]
        pages["topic_terms"] = pages.dominant_topic.apply(lambda tid: self.get_topic_terms(tid))
        pages["URL"] = pages.URL.apply(lambda u: "<a href=\"{}\">{}</a>".format(u, u.split('/')[-1]))
        pages = pages.sort_values("clicks", ascending=False)
        return pages



