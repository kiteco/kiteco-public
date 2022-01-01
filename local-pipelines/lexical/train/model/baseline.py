from typing import List, Dict, Any

import tensorflow as tf

from kite.model.model import Model
from kite.utils.aggregator import SummaryInfo
from model.feeder import Feed
from model.config import Config
from model.model import shape_list


class BaselineModel(Model):
    def loss(self) -> tf.Tensor:
        return self._loss

    def feed_dict(self, feed: Feed, train: bool) -> Dict[tf.Tensor, Any]:
        return {self._context: feed.context}

    def summary_infos(self) -> List[SummaryInfo]:
        return [SummaryInfo('accuracy')]

    def summaries_to_fetch(self) -> Dict[str, tf.Tensor]:
        return {'accuracy': self._accuracy}

    def placeholders_dict(self) -> Dict[str, tf.Tensor]:
        return {self._context.name: self._context}

    def pred(self) -> tf.Tensor:
        return self._last

    def _make_context(self):
        with tf.name_scope('placeholders'):
            # shape: (batch, context)
            self._context = tf.placeholder(
                dtype=tf.int64,
                shape=[None, None],
                name='context'
            )

    def _build_metrics_op(self):
        # shape: (batch, context)
        predicted = tf.argmax(self._pred, axis=-1)

        # shape: (batch, context - 1)
        equal_first = tf.cast(
            tf.equal(predicted[:, :-1], self._context[:, 1:]),
            tf.float32
        )

        with tf.name_scope('metrics'):
            # shape: (1,)
            self._accuracy = tf.reduce_mean(equal_first)


class UnigramModel(BaselineModel):
    def __init__(self, config: Config):
        self._make_context()

        # shape: (batch * context, 1)
        unigram = tf.reshape(self._context, shape=[-1, 1])

        # shape: (n_vocab,)
        logits = tf.get_variable(
            name='logits',
            shape=[config.n_vocab],
            initializer=tf.zeros_initializer()
        )

        # shape: (n_vocab,)
        self._probs = tf.nn.softmax(logits)

        losses = tf.log(tf.gather_nd(self._probs, unigram))

        # shape: (1,)
        self._loss = -tf.reduce_mean(losses)

        batch, sequence = shape_list(self._context)

        with tf.name_scope('prediction'):
            # shape: (batch, context, n_vocab)
            self._pred = tf.add(
                self._probs,
                tf.zeros(shape=[batch, sequence, config.n_vocab]),
                name='pred'
            )

            # shape: (batch, n_vocab)
            self._last = tf.add(
                self._probs,
                tf.zeros(shape=[batch, config.n_vocab]),
                name='last'
            )
        self._build_metrics_op()


class BigramModel(BaselineModel):
    def __init__(self, config: Config):
        self._make_context()

        # shape: (batch * (context - 1),)
        first = tf.reshape(self._context[:, :-1], shape=[-1])

        # shape: (batch * (context - 1),)
        second = tf.reshape(self._context[:, 1:], shape=[-1])

        # shape: (batch * (context - 1), 2)
        bigram = tf.stack([first, second], axis=1)

        # shape: (n_vocab, n_vocab)
        logits = tf.get_variable(
            name='logits',
            shape=[config.n_vocab] * 2,
            initializer=tf.zeros_initializer()
        )

        # shape: (n_vocab, n_vocab)
        self._probs = tf.nn.softmax(logits)

        losses = tf.log(tf.gather_nd(params=self._probs, indices=bigram))

        # shape: (1,)
        self._loss = -tf.reduce_mean(losses)

        with tf.name_scope('prediction'):
            # shape: (batch, context, n_vocab)
            self._pred = tf.gather(
                params=self._probs,
                indices=self._context,
                name='pred'
            )
            # shape: (batch, n_vocab)
            self._last = tf.gather(
                params=self._probs,
                indices=self._context[:, -1],
                name='last'
            )
        self._build_metrics_op()


class GRUModel(BaselineModel):
    def __init__(self, config: Config, units: int=128):
        self._make_context()

        # shape: (batch, context, n_embd)
        embedding = tf.keras.layers.Embedding(
            input_dim=config.n_vocab,
            output_dim=config.n_embd
        )(self._context)

        # shape: (batch, context, units)
        gru = tf.keras.layers.GRU(
            units=units,
            return_sequences=True,
            recurrent_initializer='glorot_uniform'
        )(embedding)

        # shape: (batch, context, n_vocab)
        logits = tf.keras.layers.Dense(units=config.n_vocab)(gru)

        # shape: (batch, n_vocab)
        last_logits = logits[:, -1]

        with tf.name_scope('prediction'):
            # shape: (batch, context, n_vocab)
            self._pred = tf.nn.softmax(logits, name='pred')

            #shape: (batch, n_vocab)
            self._last = tf.nn.softmax(last_logits, name='last')

        # shape: (batch, context - 1)
        losses = tf.keras.losses.sparse_categorical_crossentropy(
            y_true=self._context[:, 1:],
            y_pred=self._pred[:, :-1]
        )

        # shape: (1,)
        self._loss = tf.reduce_mean(losses)
        self._build_metrics_op()
