from typing import Dict, List, NamedTuple, Any, Tuple, Union

from enum import Enum

import tensorflow as tf
import numpy as np


class AggregateOp(Enum):
    AVG = 1
    PACK = 2
    RUNNING_TOTAL = 3


Shape = Union[np.ndarray, List, Tuple]


class SummaryInfo(NamedTuple):
    name: str
    dtype: tf.DType = tf.float32
    shape: Shape = []
    agg: AggregateOp = AggregateOp.AVG

    def get_default_scalar(self) -> Any:
        if self.dtype is tf.string:
            return 'NA'
        if self.dtype in [tf.int32, tf.int64]:
            return 0
        return 0.

    def with_name(self, name: str) -> 'SummaryInfo':
        return SummaryInfo(
            name=name,
            dtype=self.dtype,
            shape=self.shape,
            agg=self.agg,
        )


class _InfoAndPlaceholder(NamedTuple):
    placeholder: tf.Tensor
    info: SummaryInfo


class SummaryOp(NamedTuple):
        op: tf.Tensor
        infos: Dict[str, _InfoAndPlaceholder]

        @staticmethod
        def build(metrics: List[SummaryInfo]) -> 'SummaryOp':
            infos: Dict[str, _InfoAndPlaceholder] = {}
            summaries: List[tf.Summary] = []

            for metric in metrics:
                ph = tf.placeholder(metric.dtype, shape=metric.shape, name='placeholders/{}'.format(metric.name))
                infos[metric.name] = _InfoAndPlaceholder(ph, metric)
                if metric.dtype == tf.string:
                    summary = tf.summary.text(metric.name, ph)
                else:
                    summary = tf.summary.scalar(metric.name, ph)
                summaries.append(summary)

            return SummaryOp(
                op=tf.summary.merge(summaries),
                infos=infos)


class Aggregator(object):
    """
    Aggregator collects samples for named metrics at each training step, and returns a Tensorboard
    summary containing the aggregated metric values collected.
    """

    def __init__(self, summary_op: SummaryOp):
        self._summary_op = summary_op
        # metric name -> its aggregated version for the batch
        self._agg: Dict[str, Any] = {}
        self._running_sums: Dict[str, Any] = {}
        self.n_steps: int = 0

    def add(self, metrics: Dict[str, Any]):
        self.n_steps += 1
        for metric, val in metrics.items():
            info = self._summary_op.infos[metric]
            if info.info.agg is AggregateOp.AVG:
                self._agg[metric] = self._agg.get(metric, 0.) + float(val)
            else:
                self._agg[metric] = self._agg.get(metric, []) + [val]

    def get_summary(self, sess: tf.compat.v1.Session) -> tf.compat.v1.Summary:
        """
        summary returns a Tensorboard summary for averages of all the metrics, and clears the state.
        """
        feed_dict = {}
        for metric, res in self._agg.items():
            info = self._summary_op.infos[metric]
            if info.info.agg is AggregateOp.AVG:
                avg = res / float(self.n_steps)
                feed_dict[info.placeholder] = avg
            elif info.info.agg is AggregateOp.RUNNING_TOTAL:
                total = np.array(res).sum() + self._running_sums.get(metric, info.info.get_default_scalar())
                feed_dict[info.placeholder] = total
                self._running_sums[metric] = total
            else:
                feed_dict[info.placeholder] = _pack(res, info.info.dtype, info.info.get_default_scalar())

        summary = sess.run(self._summary_op.op, feed_dict=feed_dict)

        # reset the state
        self._agg.clear()
        self.n_steps = 0

        return summary


def _pack(arrs: List[np.ndarray], tf_dt: tf.DType, default: Any) -> np.ndarray:
    def _ensure_2d(a: np.ndarray) -> np.ndarray:
        if len(a.shape) == 1:
            a = a[np.newaxis, :]
        if len(a.shape) == 0:
            a = a[np.newaxis, np.newaxis]
        return a

    arr0 = _ensure_2d(arrs[0])
    max_shape = arr0.shape
    dtype = arr0.dtype.type
    assert len(max_shape) == 2, 'max shape must be rank 2, got {}'.format(max_shape)
    for i, arr in enumerate(arrs):
        arr = _ensure_2d(arr)
        arrs[i] = arr

        assert len(arr.shape) == len(max_shape), 'mismatched ranks len({}) != len({})'.format(arr.shape, max_shape)
        assert arr.dtype.type == dtype, 'mismatch dtype {} != {}'.format(arr.dtype.type, dtype)
        if np.any(arr.shape > max_shape):
            max_shape = arr.shape

    if tf_dt is not tf.string:
        new_arrs = []
        for arr in arrs:
            shape = (arr.shape[0],) + max_shape[1:]
            padded = np.full(shape, default)
            padded[0:arr.shape[0], 0:arr.shape[1]] = arr
            new_arrs.append(padded)

        return np.concatenate(new_arrs, axis=0)

    # string numpy arrays behave weird in numpy so convert to string list and then back
    new_arrs = []
    for arr in arrs:
        as_list: list = arr.tolist()
        padding = [default] * (max_shape[1] - arr.shape[1])
        for i in range(len(as_list)):
            as_list[i].extend(padding)
        new_arrs.append(np.array(as_list))

    packed = np.concatenate(new_arrs, axis=0)
    return packed
