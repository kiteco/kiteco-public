from typing import List, NamedTuple

import argparse
import datetime
import json
import logging
import tensorflow as tf

from kite.completion_mix.config import Config
from kite.completion_mix.file_feeder import FileFeederSplit
from kite.completion_mix.model import Model
from kite.completion_mix.raw_sample import RawSample
from kite.utils.serialize import serialize_namedtuple

logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(levelname)-8s %(message)s')


class VisRecord(NamedTuple):
    sample: RawSample
    probs: List[float]


def main():
    ts = str(datetime.datetime.now())

    parser = argparse.ArgumentParser()
    parser.add_argument('--traindata', type=str)
    parser.add_argument('--checkpoint_path', type=str)
    parser.add_argument('--max_samples', type=int)
    parser.add_argument('--out_path', type=str, default='out/{}/visdata'.format(ts))
    args = parser.parse_args()

    logging.info('writing visualization data results to {0}'.format(args.out_path))
    f = open(args.out_path, 'w')

    config = Config()
    split = FileFeederSplit(args.traindata, config.test_fraction)
    feeder = split.val_feeder()

    num_samples = min(args.max_samples, feeder.count())
    logging.info('will write {} samples'.format(num_samples))

    model = Model(config)

    with tf.Session() as sess:
        model.load_checkpoint(sess, args.checkpoint_path)

        for i in range(num_samples):
            sample: RawSample = feeder.next()

            feeds = model.feed_dict([sample], train=False)
            fetches = {'pred': model.pred()}
            result = sess.run(fetches, feeds)

            probs = list([float(e) for e in result['pred'].flatten()])
            record = VisRecord(sample=sample, probs=probs)
            f.write(serialize_namedtuple(record) + '\n')

    logging.info('finished writing {} records'.format(num_samples))


if __name__ == "__main__":
    main()

