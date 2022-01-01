from typing import NamedTuple, List, Dict, Any

import tensorflow as tf
import logging
import argparse
import datetime
import numpy as np
import json
import numbers

from kite.asserts.asserts import FieldValidator
from kite.model.model import TrainInputs, Config as BaseConfig, AdamTrainer, Model as BaseModel
from kite.utils.aggregator import SummaryInfo
from kite.data.line_feeder import LineFeeder


logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(levelname)-8s %(message)s')


class Feed(NamedTuple):
    logits: List[List[int]]
    labels: List[int]
    temperature_types: List[List[int]]

    @classmethod
    def from_json(cls, d: dict) -> 'Feed':
        v = FieldValidator(cls, d)
        return Feed(
            logits=[v.get_list('logits', numbers.Real)],
            labels=[v.get('label', int)],
            temperature_types=[v.get_list('temperature_types', int)],
        )

    def append(self, other: 'Feed'):
        self.logits.extend(other.logits)
        self.labels.extend(other.labels)
        self.temperature_types.extend(other.temperature_types)
    
    def validate(self):
        assert len(self.labels) == len(self.logits) == len(self.temperature_types), f'{len(self.labels)}, {len(self.logits)}, {len(self.temperature_types)}'
        for i,l in enumerate(self.logits):
            assert len(l) == len(self.logits[0]) == len(self.temperature_types[i]), f'{len(l)}, {len(self.logits[0])}, {len(self.temperature_types[i])}'



class Feeder(LineFeeder):
    @staticmethod
    def _from_lines(lines: List[str]) -> Feed:
        base = Feed(logits=[], labels=[], temperature_types=[])
        for line in lines:
            base.append(Feed.from_json(json.loads(line)))
        return base

    def next(self) -> Feed:
        return self._from_lines(super().next())

    def all(self) -> Feed:
        return self._from_lines(super().all())


class Placeholders(object):
    def __init__(self):
        with tf.name_scope('placeholders'):
            # shape [batch, vocab]
            self.logits = tf.placeholder(tf.float32, [None, None], name='logits')

            # shape [batch]
            self.labels = tf.placeholder(tf.int64, [None], name='labels')

            # shape [batch, vocab], elements are either 0 or 1
            self.temperature_types = tf.placeholder(tf.int64, [None, None], name='temperature_types')

    def feed_dict(self, feed: Feed) -> Dict[tf.Tensor, Any]:
        return {
            self.temperature_types: feed.temperature_types,
            self.labels: feed.labels,
            self.logits: feed.logits,
        }


class Model(BaseModel):
    def __init__(self, num_temperature_types: int):
        self._placeholders = Placeholders()
        self._build_model(num_temperature_types)
        self._build_loss()
        self._build_scalars()
        self._summaries = {k: v for k, v in self._scalars.items()}

    def _build_model(self, num_temperature_types: int):
        with tf.name_scope('model'):
            # shape [num_temperature_types]
            # models are typically over confident so initialize all temps to 1.5
            self._temperatures = tf.get_variable(
                name='temperatures', shape=[num_temperature_types], dtype=tf.float32,
                initializer=tf.constant_initializer(1.5 * np.ones(shape=[num_temperature_types], dtype=np.float32)),
            )

    def _build_loss(self):
        with tf.name_scope('loss'):
            # scale logits by inverse temperature
            # [batch, vocab]
            temperatures = tf.gather(self._temperatures, self._placeholders.temperature_types, name='temperatures')

            # [batch, vocab] / [batch, vocab]
            self._logits = tf.identity(
                self._placeholders.logits / temperatures, name='scaled_logits',
            )

            # NLL loss == cross entropy loss with one hot encoding for labels
            ce = tf.nn.sparse_softmax_cross_entropy_with_logits(
                labels=self._placeholders.labels, logits=self._logits, name='cross_entropy'
            )

            self._loss = tf.reduce_mean(ce, name='loss')

    def _build_scalars(self):
        with tf.name_scope('scalars'):
            # accuracy should not change, so we just add it as a sanity check
            # [batch]
            predicted = tf.argmax(self._logits, axis=-1, name='predicted')
            equal_first = tf.cast(tf.equal(predicted, self._placeholders.labels), tf.float32, name='acc_equal')
            acc = tf.reduce_mean(equal_first, name='acc')

        self._scalars = dict(acc=acc)

    def temperatures(self) -> tf.Tensor:
        return self._temperatures

    def feed_dict(self, feed: Feed, train: bool) -> Dict[tf.Tensor, Any]:
        return self._placeholders.feed_dict(feed)

    def loss(self) -> tf.Tensor:
        return self._loss

    def summary_infos(self) -> List[SummaryInfo]:
        return [SummaryInfo(k) for k in self._scalars]

    def summaries_to_fetch(self) -> Dict[str, tf.Tensor]:
        return self._summaries


def train_gradient_descent(train: str, validate: str, tensorboard: str, steps: int, num_temp_types: int) -> np.ndarray:
    model = Model(num_temperature_types=num_temp_types)

    trainer = AdamTrainer(model, BaseConfig(steps=steps, learning_rate=0.001, skip_grad_summaries=True))

    sw = tf.summary.FileWriter(tensorboard)

    train_feeder = Feeder(in_dir=train, cycle=False, batch_size=100)
    validate_feeder = Feeder(in_dir=validate, cycle=False, batch_size=100)

    with tf.Session() as sess:
        try:
            start = datetime.datetime.now()

            sess.run(tf.global_variables_initializer())
            sess.run(tf.local_variables_initializer())
            starting_step = 0

            ti = TrainInputs(session=sess, train_feeder=train_feeder, val_feeder=validate_feeder,
                             summary_writer=sw, summary_interval=10, validation_interval=30,
                             starting_step=starting_step, checkpoint_interval=int(500))

            trainer.train(ti)
            end = datetime.datetime.now()
            logging.info(f'Done training, took {end-start}')

            weights = sess.run(model.temperatures())
        finally:
            train_feeder.stop()
            validate_feeder.stop()
    return weights


def train_lbfgs(train: str, steps: int, num_temp_types: int) -> np.ndarray:
    # based on:
    # https://github.com/markdtw/temperature-scaling-tensorflow/blob/master/temp_scaling.py
    # https://github.com/gpleiss/temperature_scaling/blob/master/temperature_scaling.py

    train_feeder = Feeder(in_dir=train)

    # build one large batch for the optimizer
    batch = train_feeder.all()

    batch.validate()

    model = Model(num_temperature_types=num_temp_types)
    with tf.Session() as sess:
        try:
            start = datetime.datetime.now()

            sess.run(tf.global_variables_initializer())
            sess.run(tf.local_variables_initializer())

            fd = model.feed_dict(batch, True)

            nll = sess.run([model.loss()], feed_dict=fd)

            logging.info(f'nll before {nll}')

            optimizer = tf.contrib.opt.ScipyOptimizerInterface(
                model.loss(), options={'maxiter': steps},
            )

            optimizer.minimize(sess, feed_dict=fd)

            nll = sess.run([model.loss()], feed_dict=fd)
            logging.info(f'nll afer {nll}')

            end = datetime.datetime.now()
            logging.info(f'Done training, took {end-start}')

            weights = sess.run(model.temperatures())
        finally:
            train_feeder.stop()
    return weights


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--insearch', type=str, required=True)
    parser.add_argument('--outsearch', type=str, required=True)
    parser.add_argument('--tensorboard', type=str, default='tensorboard')
    parser.add_argument('--steps', type=int, default=1000)
    parser.add_argument('--train_samples', type=str, default='../traindata/train_samples/')
    parser.add_argument('--validate_samples', type=str, default='../traindata/validate_samples/')
    parser.add_argument('--gradient_descent', type=bool, default=False)
    parser.add_argument('--num_temperature_types', type=int, default=2)

    args = parser.parse_args()
    if args.gradient_descent:
        weights = train_gradient_descent(
            args.train_samples, args.validate_samples, args.tensorboard, args.steps, args.num_temperature_types)
    else:
        weights = train_lbfgs(args.train_samples, args.steps, args.num_temperature_types)
    logging.info(f'Done! Got weights: {weights}')

    with open(args.insearch, 'r') as f:
        old = json.load(f)
    old['IdentTemperature'] = float(weights[0])
    old['LexicalTemperature'] = float(weights[1])
    old['UseTemperatureScaling'] = True

    with open(args.outsearch, 'w') as f:
        json.dump(old, f)


if __name__ == '__main__':
    main()
