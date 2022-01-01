from typing import List, Tuple, NamedTuple

import tensorflow as tf

def shape_list(x: tf.Tensor) -> List[tf.Tensor]:
    """Deal with dynamic shape in tensorflow cleanly."""
    static = x.shape.as_list()
    dynamic = tf.shape(x)
    return [dynamic[i] if s is None else s for i, s in enumerate(static)]


def trim_padding(*, ctx: tf.Tensor, mask: tf.Tensor) -> Tuple[tf.Tensor, tf.Tensor, tf.Tensor]:
    """Trim excess padding from the beginning of ctx.

    :param ctx: shape [batch, window]
    :param mask: shape [batch, window], 0 indicates a pad token and 1 indicates a true token
    :return: (new_ctx, new_mask, num_pad)
        - new_ctx is ctx[:, min(num_pad):]
        - new_mask is mask[:, min(num_pad):]
        - num_pad tokens remaining in new_mask/new_ctx for each member of the batch, shape [batch, 1]
    """
    n_ctx = shape_list(ctx)[1]
    # NOTE: use n_ctx - reduce_sum(mask) instead of reduce_sum(1-mask) as the sub op
    # is noticeable in profiles, esp when context_mask is large & batched
    num_padding = n_ctx - tf.cast(tf.reduce_sum(mask, axis=1, keepdims=True), tf.int32)
    min_padding = tf.reduce_min(num_padding)
    new_ctx = ctx[:, min_padding:]
    new_mask = mask[:, min_padding:]
 
    # update post trimming
    num_padding = num_padding - min_padding

    return new_ctx, new_mask, num_padding


def repeat(*, input: tf.Tensor, num: tf.Tensor, axis: tf.Tensor=0) -> tf.Tensor:
    """
    Like tf.repeat but repeats all inputs num times along axis
    :param num: scalar tensor
    :return: shape [num*input.shape[0]] + input.shape[1:]
    """
    # TODO: use tf.repeat once we upgrade to 1.15 or later
    with tf.name_scope('build_batch_indices'):
        d = shape_list(input)[axis]
        # [batch, 1]
        batch_idxs = tf.expand_dims(tf.range(0, d), axis=-1)
        # [batch, width]
        batch_idxs = tf.tile(batch_idxs, [1, num])
        # [batch*width]
        batch_idxs = tf.reshape(batch_idxs, [-1])
    return tf.gather(input, batch_idxs, axis=axis)

    # d = shape_list(input)[axis]
    # repeats = tf.compat.v1.repeat(num, d)
    # return tf.compat.v1.repeat(input, repeats, axis=axis)

def concat(values, axis, name=None):
    """
    Requires len(values) == 2
    Equivalent to tf.concat([first, second], axis=axis), but 
        - if values[0].shape[axis] == 0 (values[1].shape[axis] == 0) then return values[1] (values[0])
        - if values[0].shape[axis] == 0 and values[0].shape[axis] == 0 then return tf.concat(values, axis=axis)
        - if values[0].shape[axis] != 0 and values[0].shape[axis] != 0 then return tf.concat(values, axis=axis)
    """
    assert len(values) == 2
    first, second = values

    ns = f'concat_{name}' if name else 'concat'
    with tf.name_scope(ns):
        d1 = shape_list(first)[axis]
        d2 = shape_list(second)[axis]

        num_non_zero = tf.math.count_nonzero([d1, d2])
        def a_zero():
            return tf.cond(
              tf.equal(d1, 0),
              true_fn=lambda: second,
              false_fn=lambda: first,
            )
        
        res = tf.cond(
          tf.equal(num_non_zero, 1),
          true_fn=a_zero,
          false_fn=lambda: tf.concat([first, second], axis=axis),
        )
    return tf.identity(res, name=name)


