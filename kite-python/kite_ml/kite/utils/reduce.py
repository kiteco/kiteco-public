import tensorflow as tf


def is_empty(x: tf.Tensor) -> tf.Tensor:
    return tf.equal(tf.reduce_sum(tf.shape(x)), 0)


def safe_reduce_mean(x: tf.Tensor, value: float, name: str) -> tf.Tensor:
    # need conditional in case the tensor is empty to avoid nans
    with tf.name_scope('{}_safe_mean'.format(name)):
        return tf.cond(
            is_empty(x),
            true_fn=lambda: value, false_fn=lambda: tf.reduce_mean(x), name=name,
        )