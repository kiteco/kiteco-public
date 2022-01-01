import logging
import argparse
import datetime
import json
import os
from typing import NamedTuple, Dict, Any, List, Tuple, Optional

import tensorflow as tf

horovod_enabled = True
try:
    import horovod.tensorflow as hvd
except ImportError:
    horovod_enabled = False

from kite.utils.save import save_model
from kite.model.model import Model, TrainInputs, Config as BaseConfig, _Trainer as BaseTrainer, AdamTrainer, DataFeeder
from kite.utils.aggregator import SummaryInfo

from model.prefix_suffix_lm import Model as PrefixSuffixLM
from model.model import LexicalModel
from model.config import Config
from model.feeder import FileDataFeeder, Feed
from model.baseline import UnigramModel, BigramModel, GRUModel

logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(levelname)-8s %(message)s')


class HorovodAdamTrainer(BaseTrainer):
    def __init__(self, model: Model, config: Config):
        self._config : Config = config
        self._lr : tf.Tensor = tf.compat.v1.placeholder_with_default(config.learning_rate, shape=[], name='learning_rate')
        self._optimizer:  tf.compat.v1.train.Optimizer = tf.compat.v1.train.AdamOptimizer(learning_rate=self._lr)

        self._scalars = ['learning_rate']
        self._summaries = {'learning_rate' : self._lr}

        # Go ahead and wrap our optimizer in hvd.DistributedOptimizer. This works fine w/ 1 GPU as well
        # NOTE: Do this BEFORE calling super().__init__ -- the optimizer is used to build the training op
        # in BaseTrainer
        self._optimizer = hvd.DistributedOptimizer(self._optimizer)

        super().__init__(model, config)

    def optimizer(self) -> tf.compat.v1.train.Optimizer:
        return self._optimizer

    def distributed(self) -> bool:
        return True

    def broadcast_global_variables(self, session: tf.compat.v1.Session):
        session.run(hvd.broadcast_global_variables(0))

    def rank(self) -> int:
        return hvd.local_rank()

    def optimizer_feeds(self, global_step) -> Dict[tf.Tensor, Any]:
        return {
            self._lr: self._learning_rate(global_step),
        }

    def summary_infos(self) -> List[SummaryInfo]:
        return [SummaryInfo(k) for k in self._scalars]

    def summaries_to_fetch(self) -> Dict[str, tf.Tensor]:
        return self._summaries

    def _learning_rate(self, global_step) -> float:
        linear = ((1. - self._config.starting_rate) / self._config.warmup_steps) * global_step \
            + self._config.starting_rate
        decay = self._config.warmup_steps ** .5 / global_step ** .5

        if global_step <= self._config.warmup_steps:
            return linear*self._config.learning_rate

        return decay*self._config.learning_rate


def save(model: Model, sess: tf.compat.v1.Session, outdir: str):
    inputs = model.placeholders_dict()
    outputs = model.outputs_dict()

    save_model(
        sess,
        outdir,
        inputs=inputs,
        outputs=outputs,
    )


def get_model(config):
    models = {
        'unigram': UnigramModel,
        'bigram': BigramModel,
        'gru': GRUModel,
        'lexical': LexicalModel,
        'prefix_suffix': PrefixSuffixLM,
    }
    Model = models[config.model_type]
    return Model(config)


def main():
    if horovod_enabled:
        hvd.init()
        print(f"horovod, local_rank: {hvd.local_rank()}, rank: {hvd.rank()}")

    parser = argparse.ArgumentParser()
    parser.add_argument('--out_dir', type=str, default='out')
    parser.add_argument('--tensorboard', type=str, default='tensorboard')
    parser.add_argument('--steps', type=int, default=100)
    parser.add_argument('--checkpoint_path', type=str, default='tmp/ckpt')
    parser.add_argument('--train_samples', type=str, default='out/train_samples')
    parser.add_argument('--validate_samples', type=str, default='out/validate_samples')
    parser.add_argument('--config', type=str, default='')
    parser.add_argument('--train_batch_size', type=int, default=20)
    parser.add_argument('--validate_batch_size', type=int, default=20)
    parser.add_argument('--load_model', type=str, default='')
    parser.add_argument('--from_checkpoint', type=bool, default=False)
    parser.add_argument('--resume_model', type=str, default='')
    parser.add_argument('--resume_steps', type=int, default=0)

    args = parser.parse_args()

    config = Config()
    if args.config != '':
        config = Config.from_json(json.load(open(args.config, 'r')))

    # Custom configuration to:
    # - Allow GPU usage to grow as needed vs preallocating
    tf_config = tf.compat.v1.ConfigProto()
    tf_config.gpu_options.allow_growth = True

    tensorboard_path = args.tensorboard

    rank = 0
    size = 1

    model = get_model(config)

    warmup_steps = int(args.steps * 0.20) # warmup should be 20% of all steps
    base_config = BaseConfig(steps=args.steps, warmup_steps=warmup_steps, learning_rate=0.001, skip_grad_summaries=True)

    if horovod_enabled:
        rank = hvd.local_rank()
        size = hvd.size()

        # Add `_hvdN` suffix here so that all mirrors can output to, and be visible from tensorboard
        tensorboard_path = args.tensorboard + "_hvd" + str(hvd.rank())
        # Pin process to a single GPU
        tf_config.gpu_options.visible_device_list = str(hvd.local_rank())
        trainer = HorovodAdamTrainer(model, base_config)
    else:
        trainer = AdamTrainer(model, base_config)

    sw = tf.compat.v1.summary.FileWriter(tensorboard_path)

    train_feeder = FileDataFeeder(args.train_samples, batch_size=args.train_batch_size, shard=rank, num_shards=size)
    validate_feeder = FileDataFeeder(args.validate_samples, batch_size=args.validate_batch_size, shard=rank, num_shards=size)

    with tf.compat.v1.Session(config=tf_config) as sess:
        try:
            start = datetime.datetime.now()
            if args.from_checkpoint:
                saver = tf.train.Saver()

                # Load the most recent model from a checkpoint path, and get the current global step
                ckpt = tf.train.get_checkpoint_state(os.path.dirname(args.checkpoint_path))
                if ckpt and ckpt.model_checkpoint_path:
                    saver.restore(sess, ckpt.model_checkpoint_path)

                # Get the step from the filename. Kinda hacky.
                starting_step = int(os.path.basename(ckpt.model_checkpoint_path).split('-')[1]) + 1
                logging.info(f"Resuming model from checkpoint {args.checkpoint_path} from step {starting_step}")
            elif args.resume_model != "":
                variables_to_restore = [var for var in tf.compat.v1.global_variables() if "/Adam" not in var.name and "train/" not in var.name]
                variables_to_init = [var for var in tf.compat.v1.global_variables() if "/Adam" in var.name or "train/" in var.name]

                saver = tf.compat.v1.train.Saver(var_list=variables_to_restore)
                saver.restore(sess, os.path.join(args.resume_model, "variables/variables"))
                sess.run(tf.initialize_variables(variables_to_init))

                starting_step = args.resume_steps
                logging.info(f"Resuming model from {args.resume_model}")
            elif args.load_model != "":
                variables_to_restore = [var for var in tf.compat.v1.global_variables() if "/Adam" not in var.name and "train/" not in var.name]
                variables_to_init = [var for var in tf.compat.v1.global_variables() if "/Adam" in var.name or "train/" in var.name]

                saver = tf.compat.v1.train.Saver(var_list=variables_to_restore)
                saver.restore(sess, os.path.join(args.load_model, "variables/variables"))
                sess.run(tf.initialize_variables(variables_to_init))

                starting_step = 0
                logging.info(f"Loading model from {args.load_model}")
            else:
                sess.run(tf.compat.v1.global_variables_initializer())
                sess.run(tf.compat.v1.local_variables_initializer())

                if horovod_enabled:
                    # Broadcast variables after initialization to avoid divergence from
                    # different random initializations
                    trainer.broadcast_global_variables(sess)

                starting_step = 0

            if horovod_enabled:
                ti = TrainInputs(session=sess, train_feeder=train_feeder, val_feeder=validate_feeder,
                                summary_writer=sw, checkpoint_save_path=args.checkpoint_path,
                                summary_interval=10, validation_interval=30,
                                starting_step=starting_step, checkpoint_interval=int(500))
            else:
                ti = TrainInputs(session=sess, train_feeder=train_feeder, val_feeder=validate_feeder,
                                summary_writer=sw, checkpoint_save_path=args.checkpoint_path,
                                summary_interval=10, validation_interval=30, validation_based_checkpoint=True,
                                starting_step=starting_step, checkpoint_interval=int(500))

            trainer.train(ti)
            end = datetime.datetime.now()
            logging.info('Done training, took {0}'.format(end-start))

            # Only mirror 0 saves the model to avoid corruption
            if rank == 0:
                save(model, sess, args.out_dir)

        except KeyboardInterrupt:
            print('interrupted!')
            if rank == 0:
                print('saving model...')
                save(model, sess, args.out_dir)
        finally:
            train_feeder.stop()
            validate_feeder.stop()


if __name__ == '__main__':
    main()
