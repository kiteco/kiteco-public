import json

import pandas as pd
import numpy as np


import sklearn
from sklearn.linear_model import LinearRegression
from sklearn.multioutput import MultiOutputRegressor
from sklearn.ensemble import RandomForestRegressor

from sklearn.preprocessing import StandardScaler
from sklearn.pipeline import Pipeline


# Rate got from https://www.advancedwebranking.com/ctrstudy/
ctr_rate = [30.61,15.84,10.21,6.47,4.37,3.05,2.23,1.7,1.34,1.08,1,1.37,1.71,1.39,1.22,1.11,1.14,1.15,1.19,1]

class TrafficEstimator(object):

    def __init__(self, gsc_data, moz_data, so_data, split_test_ratio = 0.1, model_to_train=LinearRegression()):
        self.gsc_data = gsc_data
        self.moz_data = moz_data
        self.so_data = so_data
        self.features = None
        self.label = None
        self.training_dataset = None
        self.dataset = None
        self.split_test_ratio = split_test_ratio
        self.prepare_dataset()
        self.model, self.score = self.train_model(model_to_train)
        self.predictions = self.predict_volume()


    def prepare_dataset(self):
        self.prepare_gsc_data()
        self.prepare_moz_data()
        self.dataset = self.moz_data.reset_index(drop=True).merge(self.gsc_data.reset_index(drop=True), 'inner', on='keywords')
        if len(self.dataset) < 500:
            raise ValueError("Not enough value to train the model after merging Moz and GSC data, we should have "
                             "at least 500 points after the merge and we only have {} points. Please check the datasets"
                             .format(len(self.dataset)))
        self.training_dataset = self.dataset[['moz_difficulty', 'moz_volume',
                              'so_rank', 'volume_ratio', 'est_clicks', 'est_impressions', 'clicks',
                              'ctr', 'impressions', 'position']].copy()
        self.features = ['moz_difficulty', 'moz_volume', 'so_rank', 'volume_ratio', 'est_clicks', 'est_impressions']
        self.label = 'impressions'
        self.training_dataset["is_test"] = np.random.random(len(self.training_dataset)) < self.split_test_ratio


    def prepare_moz_data(self):
        self.moz_data = self.moz_data[['URL', 'keywords', 'moz_difficulty', 'moz_volume', 'question_id', 'so_rank']]
        per_questions = self.moz_data.groupby("question_id").moz_volume.sum()
        per_questions = per_questions.reset_index().set_index("question_id")
        per_questions = per_questions.set_index(per_questions.index.astype(int))

        def get_question_volume(id):
            return per_questions.loc[id].iloc[0]

        self.moz_data["volume_ratio"] = self.moz_data.apply(lambda r: r.moz_volume / get_question_volume(int(r.question_id)), axis=1)

        questions_views = self.so_data.newViews

        def get_question_clicks(id):
            if id in questions_views.index:
                return questions_views.loc[id].iloc[0]
            else:
                return -1

        self.moz_data["est_clicks"] = self.moz_data.apply(lambda r: r.volume_ratio*get_question_clicks(int(r.question_id)), axis=1)
        self.moz_data = self.moz_data[self.moz_data.est_clicks > 0].copy()

        def get_ctr(rank):
            if rank > len(ctr_rate) or np.isnan(rank):
                rank = len(ctr_rate)
            return ctr_rate[int(rank-1)]/100

        self.moz_data["est_impressions"] = self.moz_data.est_clicks/self.moz_data.so_rank.apply(get_ctr)


    def prepare_gsc_data(self):
        self.gsc_data['keywords'] = self.gsc_data["keys"].apply(lambda k : k[0])
        self.gsc_data['URL'] = self.gsc_data['keys'].apply(lambda k : k[1])
        self.gsc_data.drop(columns=["keys"], inplace=True)

        def get_query_features(q):
            url = q.loc[q['impressions'].idxmax()].URL
            impressions = q.impressions.sum()
            clicks = q.clicks.sum()
            ctr = clicks/impressions
            position = q.position.min()
            return pd.Series(dict(
                clicks=clicks,
                ctr=ctr,
                impressions=impressions,
                position=position,
                keywords=q.name,
                URL=url
            ))

        self.gsc_data = self.gsc_data.groupby('keywords').apply(get_query_features)
        self.gsc_data = self.gsc_data[self.gsc_data.URL.apply(lambda u : "/examples/" in u)].copy()
        self.gsc_data = self.gsc_data.nlargest(5000, ["clicks"], "all")

    def train_model(self, model=LinearRegression()):
        test_data = self.training_dataset[self.training_dataset.is_test]
        train_data = self.training_dataset[~self.training_dataset.is_test]

        scaler = StandardScaler()
        pipeline = Pipeline(steps=[("Scaler", scaler), ("Model", model)])
        pipeline.fit(train_data[self.features], train_data[[self.label]])
        score = pipeline.score(test_data[self.features], test_data[self.label])
        print("{} R2 score : {}".format(self.label, score))

        return pipeline, score

    def predict_volume(self):
        self.moz_data["kite_predicted_impressions"] = self.model.predict(self.moz_data[self.features])
        predicted_impressions = self.moz_data.groupby("URL").kite_predicted_impressions.sum()
        predicted_impressions.reset_index().set_index("URL")
        return predicted_impressions
