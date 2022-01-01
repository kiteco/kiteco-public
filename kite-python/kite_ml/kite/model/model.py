from typing import NamedTuple, Dict, Any, List, Tuple, Optional

import tensorflow as tf

import os

import logging

import datetime

from ..utils.aggregator import Aggregator, SummaryOp, SummaryInfo

logger = logging.getLogger(__name__)


class Model(object):
    def loss(self) -> tf.Tensor:
        """
        loss op
        :return:
        """
        raise NotImplementedError('loss must be implemented by subclasses')

    def feed_dict(self, sample: Any, train: bool) -> Dict[tf.Tensor, Any]:
        """
        feed dict for a run
        :param sample: sample to build feed dict from, taken from the feeder
        :param train: true if this is for a training run
        :return: feed dict for a run
        """
        raise NotImplementedError('feed_dict must be implemented by subclasses')

    def summary_infos(self) -> List[SummaryInfo]:
        """
        summary infos to aggregate
        """
        raise NotImplementedError('summary_infos must be implemented by subclasses')

    def summaries_to_fetch(self) -> Dict[str, tf.Tensor]:
        """
        summaries to fetch for a run, should NOT include training op or loss op
        :return: summaries to fetch for a run
        """
        raise NotImplementedError('summaries_to_fetch must be implemented by subclasses')


class DataFeeder(object):
    """DataFeeder defines an abstract class that iterates over records of any type."""
    def next(self) -> Any:
        raise NotImplementedError("subclasses of DataFeeder must implement next")

    def stop(self):
        raise NotImplementedError("subclasses of DataFeeder must implement stop")


class Config(NamedTuple):
    clip_gradients: bool = False
    max_gradient_norm: float = 1.
    learning_rate: float = 1e-4
    warmup_steps: int = 50000
    starting_rate: float = 0.35 # ratio of learning_rate to start at step 0
    steps: int = int(1e6)
    skip_grad_summaries: bool = False


class Result(NamedTuple):
    loss: float
    summaries: Dict[str, Any]


class TrainInputs(NamedTuple):
    session: tf.compat.v1.Session
    train_feeder: DataFeeder
    val_feeder: DataFeeder
    summary_writer: tf.compat.v1.summary.FileWriter
    checkpoint_save_path: str
    starting_step: int
    validation_interval: int = 50
    validation_based_checkpoint: bool = False
    validation_smoothing_weight: float = 0.8
    checkpoint_interval: int = 50000
    # summary_interval defines the interval at which scalars get averaged and sent to tensorboard.
    summary_interval: int = 50
    broadcast_interval: int = 50000


class _Trainer(object):
    def optimizer(self) -> tf.compat.v1.train.Optimizer:
        """
        optimizer to use
        :return: optimizer to use
        """
        raise NotImplementedError('sub classes must implement optimizer')

    def distributed(self) -> bool:
        """
        :return: whether this trainer is distribued aware.
        """
        raise NotImplementedError('sub classes must implement optimizer')

    def broadcast_global_variables(self, session: tf.compat.v1.Session):
        """
        implement brodcast behavior to sync global variables between replicas.
        only used in the horovod trainer.
        """
        raise NotImplementedError('sub classes must implement broadcast_global_variables')

    def rank(self) -> int:
        """
        return the rank of the current replica
        """
        raise NotImplementedError('sub classes must implement rank')

    def optimizer_feeds(self, global_step) -> Dict[tf.Tensor, Any]:
        """
        return optimizer-specific feed dict
        """
        raise NotImplementedError('sub classes must implement optimizer_feeds')

    def summary_infos(self) -> List[SummaryInfo]:
        """
        summary infos to aggregate
        """
        raise NotImplementedError('summary_infos must be implemented by subclasses')

    def summaries_to_fetch(self) -> Dict[str, tf.Tensor]:
        """
        summaries to fetch for a run, should NOT include training op or loss op
        :return: summaries to fetch for a run
        """
        raise NotImplementedError('summaries_to_fetch must be implemented by subclasses')

    def __init__(self, model: Model, config: Config):
        self._model = model
        self._config = config
        self._build()

    def _build(self):
        self._add_train_op()
        self._build_grad_scalars()

    def _add_train_op(self):
        with tf.name_scope('train'):
            optimizer = self.optimizer()

            clipped: List[Tuple[Optional[tf.Tensor], tf.Variable]] = []
            grads: List[Tuple[tf.Tensor, tf.Variable]] = []
            for grad, var in optimizer.compute_gradients(self._model.loss()):
                if grad is not None:
                    if self._config.clip_gradients:
                        grad = tf.clip_by_norm(grad, self._config.max_gradient_norm)
                    clipped.append((grad, var))
                    grads.append((grad, var))
                else:
                    clipped.append((grad, var))
            self._grads = grads
            self._train_op = optimizer.apply_gradients(clipped)

    def _build_grad_scalars(self):
        if self._config.skip_grad_summaries:
            self._grad_scalars = {}
            return

        scalars: Dict[str, tf.Tensor] = {}
        for grad, var in self._grads:
            name = "norm_gradient_wrt_{}".format(var.name).replace(":", "_")
            scalars[name] = tf.norm(grad)
        self._grad_scalars = scalars

    def _run(self, sess: tf.compat.v1.Session, global_step: int, sample: Any, train: bool) -> Result:
        feeds = dict()
        feeds.update(self._model.feed_dict(sample, train))

        summaries_to_fetch = dict()
        summaries_to_fetch.update(self._model.summaries_to_fetch())

        fetches = {'loss': self._model.loss()}

        if train:
            feeds.update(self.optimizer_feeds(global_step))
            summaries_to_fetch.update(self.summaries_to_fetch())
            summaries_to_fetch.update(self._grad_scalars)
            fetches['train'] = self._train_op

        for metric, scalar in summaries_to_fetch.items():
            fetches['summary/' + metric] = scalar

        result = sess.run(fetches, feeds)

        summaries = {metric: result['summary/' + metric]
                     for metric in summaries_to_fetch.keys()}

        summaries['loss'] = result['loss']

        return Result(loss=result['loss'], summaries=summaries)

    def train(self, ti: TrainInputs):
        ti.summary_writer.add_graph(ti.session.graph)
        # initialize saver and make sure its save directory exists
        os.makedirs(os.path.dirname(ti.checkpoint_save_path), exist_ok=True)
        saver = tf.train.Saver(max_to_keep=1)

        summary_info = self._model.summary_infos() + [SummaryInfo('loss')]
        with tf.name_scope('train_summary'):
            grad_info = [SummaryInfo(k) for k in self._grad_scalars.keys()]
            train_info = summary_info + grad_info + self.summary_infos()
            train_aggregator = Aggregator(SummaryOp.build(train_info))
        with tf.name_scope('validate_summary'):
            val_aggregator = Aggregator(SummaryOp.build(summary_info))

        logging.info("Running training for {} steps".format(self._config.steps))
        max_validation_acc = 0
        moving_validation_acc = 0
        for step in range(ti.starting_step, ti.starting_step + self._config.steps):
            start = datetime.datetime.now()
            train_res = self._run(ti.session, step+1, ti.train_feeder.next(), train=True)
            duration = datetime.datetime.now() - start

            logging.info("step {}: loss {}, took {}".format(step, train_res.loss, duration))
            train_aggregator.add(train_res.summaries)

            if step > 0 and step % ti.validation_interval == 0:
                start = datetime.datetime.now()
                validate_res = self._run(ti.session, step+1, ti.val_feeder.next(), train=False)
                duration = datetime.datetime.now() - start
                logging.info("validation {}, loss {}, took {}".format(step, validate_res.loss, duration))

                if ti.validation_based_checkpoint:
                    if moving_validation_acc == 0:
                        moving_validation_acc = validate_res.summaries['accuracy']
                    else:
                        moving_validation_acc = ti.validation_smoothing_weight * moving_validation_acc + \
                                             (1 - ti.validation_smoothing_weight) * validate_res.summaries['accuracy']
                    if moving_validation_acc > max_validation_acc:
                        logging.info('check pointing model at step {} with smoothed validation accuracy {}'.
                                     format(step, moving_validation_acc))
                        saver.save(ti.session, ti.checkpoint_save_path, write_meta_graph=False, global_step=step)
                        logging.info('model saved to {}'.format(ti.checkpoint_save_path))
                        max_validation_acc = moving_validation_acc
                    else:
                        logging.info('skipping check pointing model at step {} with smoothed validation accuracy {}'.
                                     format(step, moving_validation_acc))

                val_aggregator.add(validate_res.summaries)

            # periodically broadcast variables to keep mirrors in sync
            if self.distributed() and step > 0 and step % ti.broadcast_interval == 0:
                start = datetime.datetime.now()
                self.broadcast_global_variables(ti.session)
                duration = datetime.datetime.now() - start
                logging.info("broadcast global variables at step {}, took {}".format(step, duration))

            if self.rank() == 0 and not ti.validation_based_checkpoint and \
                    (step == 0 or (step + 1) % ti.checkpoint_interval == 0):
                logging.info('check pointing the model at step {}'.format(step))
                saver.save(ti.session, ti.checkpoint_save_path, write_meta_graph=False, global_step=step)
                logging.info('model saved to {}'.format(ti.checkpoint_save_path))

            if step > 0 and step % ti.summary_interval == 0:
                for agg in [train_aggregator, val_aggregator]:
                    if agg.n_steps > 0:
                        summary = agg.get_summary(ti.session)
                        ti.summary_writer.add_summary(summary, step)


class AdamTrainer(_Trainer):
    def __init__(self, model: Model, config: Config):
        self._optimizer:  tf.compat.v1.train.Optimizer = tf.compat.v1.train.AdamOptimizer(learning_rate=config.learning_rate)
        super().__init__(model, config)

    def optimizer(self) -> tf.compat.v1.train.Optimizer:
        return self._optimizer

    def distributed(self) -> bool:
        return False

    def rank(self) -> int:
        return 0

    def broadcast_global_variables(self, session: tf.compat.v1.Session):
        pass

    def optimizer_feeds(self, global_step) -> Dict[tf.Tensor, Any]:
        return {}

    def summary_infos(self) -> List[SummaryInfo]:
        return []

    def summaries_to_fetch(self) -> Dict[str, tf.Tensor]:
        return {}