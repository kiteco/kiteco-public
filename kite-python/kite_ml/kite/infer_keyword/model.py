import json
from typing import Callable, List

import tensorflow as tf
import tqdm
import time
import random
import numpy as np

from .classifier import SoftmaxClassifier
from .config import Config
from .constants import Constants
from .data import Batch, DataFeeder, Dataset
from .features import FeatureEncoder


class TrainInputs(object):
    def __init__(self,
                 config: Config,
                 sess: tf.Session,
                 train: Dataset,
                 test: Dataset,
                 classifier: SoftmaxClassifier,
                 label_selector: Callable[[Batch], List[int]]):
        self.sess = sess
        self.train = train
        self.test = test
        self.classifier = classifier
        self.label_selector = label_selector
        self.batch_size = config.batch_size
        self.n_epochs = config.n_epochs


class Model(object):
    def __init__(self, feature_encoder: FeatureEncoder):
        self.feature_encoder = feature_encoder

        with tf.name_scope('features'):
            self.features_size: int = feature_encoder.in_size()

            # features has a size of [batch size, features size]
            self.features: tf.Tensor = tf.placeholder(tf.int64, [None, self.features_size], name='features')

            # x has a size of [batch size, encoded features size]
            self.x: tf.Tensor = feature_encoder.encode_op(self.features, name='x')

        with tf.name_scope('classifiers'):
            # is_keyword has a size of [batch size, 2]
            self.is_keyword = SoftmaxClassifier(self.x, 2, "is_keyword")
            # which_keyword has a size of [batch size, number of keyword classes]
            self.which_keyword = SoftmaxClassifier(self.x, Constants.N_KEYWORDS, "which_keyword")

    # train_classifier trains the given classifier and returns the final accuracy
    def train_classifier(self, inputs: TrainInputs, stats: list = None, baseStat: dict = None) -> float:
        print("training for {0} epochs".format(inputs.n_epochs))

        accuracy = 0.
        for epoch in range(inputs.n_epochs):
            start = time.time()
            feeder = DataFeeder(inputs.train, self.feature_encoder, inputs.batch_size)

            total_loss = 0.

            for batch in tqdm.tqdm(feeder, total=feeder.n_batches):  # type: Batch
                fetches = [inputs.classifier.optimizer, inputs.classifier.loss]
                feeds = {
                    self.features: batch.features,
                    inputs.classifier.labels: inputs.label_selector(batch)
                }
                _, loss = inputs.sess.run(fetches, feed_dict=feeds)
                total_loss += loss


            test_batch = Batch(inputs.test.records, self.feature_encoder)
            test_feed_dict = {
                self.features: test_batch.features,
                inputs.classifier.labels: inputs.label_selector(test_batch),
            }
            accuracy, logits = inputs.sess.run([inputs.classifier.accuracy, inputs.classifier.logits], feed_dict=test_feed_dict)

            print("Epoch: {epoch}/{n_epochs}, avg cost: {avg_cost}, test accuracy: {accuracy}".format(
                epoch=epoch+1,
                n_epochs=inputs.n_epochs,
                avg_cost=total_loss / feeder.n_batches,
                accuracy=accuracy))
            total_time = time.time() - start
            if stats is not None:
                # The conf matrix is always computed with the keyword_cat as first arg (even for isKeyword model).
                # That allows to have the stats for isKeyword for each keyword.
                # Just have to sum over all the keywords to have the global stats.
                conf_matrix = tf.confusion_matrix(test_batch.keyword_cat, tf.argmax(logits, 1)).eval(session=inputs.sess).tolist()
                if baseStat['training'] == 'isKeyword':
                    conf_matrix = np.array(conf_matrix)[:, :2].tolist()
                newStat = dict(time=total_time, accuracy=accuracy.item(), epoch=epoch+1, n_epochs=inputs.n_epochs,
                               features_count=self.features_size, avg_cost=total_loss / feeder.n_batches,
                               total_loss=total_loss, conf_matrix=conf_matrix)
                if baseStat:
                    newStat.update(baseStat)
                stats.append(newStat)
                best = inputs.sess.run([tf.argmax(logits, 1)],test_feed_dict)
                if epoch == inputs.n_epochs-1:
                    newStat['examples'] = self.extract_examples(target_keywords=list(range(Constants.N_KEYWORDS)),
                                                                prediction=best, raw_records=inputs.test.records)

        return accuracy

    def extract_examples(self, target_keywords: list, prediction: list, raw_records: list):
        examples = {}
        max_count = 50
        sampling_rate = 1
        for kw in target_keywords:
            good_pred = []
            bad_pred = []
            for pred, rec in zip(prediction[0], raw_records):
                if rec.keyword_cat == kw+1:
                    if pred == kw:
                        if len(good_pred) < max_count and random.random() < sampling_rate:
                            good_pred.append({"data": json.dumps(rec.features.__dict__), "pred": pred.tolist()})
                    else:
                        if(len(bad_pred) < max_count) and random.random() < sampling_rate:
                            bad_pred.append({"data": json.dumps(rec.features.__dict__), "pred": pred.tolist()})
            examples[kw] = {"good_pred": good_pred, "bad_pred": bad_pred}
        return examples
