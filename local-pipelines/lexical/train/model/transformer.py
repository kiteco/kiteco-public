from typing import List

import numpy as np
import tensorflow as tf

from .config import Config
from .util import shape_list, repeat

def tfprint(tensor: tf.Tensor, name='') -> tf.Tensor:
    if name == '':
        name = tensor.name
    return tf.Print(tensor, [tensor], message="\n"+name+": ", summarize=-1)


def gelu(x):
    """
    Implements Gaussian Error Linear Unit activaton ala https://arxiv.org/abs/1606.08415.
    :param x:
    :return: has shape == x.shape
    """
    return 0.5*x*(1+tf.tanh(np.sqrt(2/np.pi)*(x+0.044715*x*x*x)))


def layer_norm(x, scope, *, axis=-1, epsilon=1e-5):
    """
    Normalize the rows of x to have to mean = 0, std = 1, then apply a diagonal affine transform to each row.
    :param x:
    :param scope:
    :param axis: axis to perform operations along
    :param epsilon:
    :return: has shape == x.shape
    """
    with tf.compat.v1.variable_scope(scope):
        n_state = x.shape[-1].value
        g = tf.compat.v1.get_variable('g', [n_state], initializer=tf.constant_initializer(1))
        b = tf.compat.v1.get_variable('b', [n_state], initializer=tf.constant_initializer(0))
        u = tf.reduce_mean(x, axis=axis, keepdims=True)
        s = tf.reduce_mean(tf.square(x-u), axis=axis, keepdims=True)
        x = (x - u) * tf.compat.v1.rsqrt(s + epsilon)
        x = x*g + b
        return x


def split_states(x, n):
    """Reshape the last dimension of x into [n, x.shape[-1]/n]."""
    *start, m = shape_list(x)
    return tf.reshape(x, start + [n, m//n])


def merge_states(x):
    """Smash the last two dimensions of x into a single dimension."""
    *start, a, b = shape_list(x)
    return tf.reshape(x, start + [a*b])


def conv1d(x, scope, nf, *, w_init_stdev=0.02, cpu_or_train=True):
    """
    Applies an Affine transform to the last axis of x.
    :param x:
    :param scope:
    :param nf: dimension of last axis of returned value
    :param w_init_stdev:
    :return: transformed x of shape x.shape[:-1] + [nf]
    """
    with tf.compat.v1.variable_scope(scope):
        *start, nx = shape_list(x)
        w = tf.compat.v1.get_variable('w', [1, nx, nf], initializer=tf.random_normal_initializer(stddev=w_init_stdev))
        b = tf.compat.v1.get_variable('b', [nf], initializer=tf.constant_initializer(0))
        if cpu_or_train:
            # [batch, context, nf]
            c = tf.nn.conv1d(x, filters=w, stride=1, padding='VALID') + b
        else:
            # NOTE: this is memory intensive for large embeddings, but if it fits on the GPU its faster
            c = tf.reshape(tf.matmul(tf.reshape(x, [-1, nx]), tf.reshape(w, [-1, nf]))+b, start+[nf])
        return c


def lr_mask(nd, ns, dtype, *, batch=None):
    """ Left right causal mask, 1's in the lower triangle, counting from the lower right corner.

    Same as tf.matrix_band_part(tf.ones([nd, ns]), -1, ns-nd), but doesn't produce garbage on TPUs.
    :return: shape [nd, ns] or [batch, nd, ns]
    """
    i = tf.range(nd)[:, None]
    j = tf.range(ns)
    m = i >= j - ns + nd
    m = tf.cast(m, dtype)
    if batch is not None:
        m = repeat(input=tf.expand_dims(m, axis=0), num=batch)
    return m



def attn(x, scope, n_state, *, mask: tf.Tensor, config: Config, past=None, cpu_or_train=True):
    """
    Performs the following operations on x to produce h:
      - affine transformation
      - multi headed attention (with masking)
      - affine transformation
    :param x: shape [batch, num_dest_tokens, embedding]
    :param n_state: dimension of transformed outputs
    :mask: Should already account for past,
           shape [batch, num_dest_tokens, num_src_tokens=num_dest_tokens+num_past_tokens],
           or shape [num_dest_tokens, num_src_tokens].
    :param past: cached intermediate state, shape [batch, 2, heads, num_past_tokens, embedding // heads]
    :return: (h, present) where
             - h is the transformed state with shape x.shape[:-1] + [n_state]
             - present.shape == [batch, 2, heads, num_dest_tokens, embedding // heads] is the intermediate state
               needed to compute h_new for some x_new conditioned on the current x.
    """
    assert x.shape.ndims == 3
    assert n_state % config.n_head == 0
    assert len(shape_list(mask)) == 2 or len(shape_list(mask)) == 3

    def split_heads(x):
        # From [batch, num_dest_tokens, embedding] to [batch, heads, num_dest_tokens, embedding // heads]
        return tf.transpose(split_states(x, config.n_head), [0, 2, 1, 3])

    def merge_heads(x):
        # Reverse of split_heads
        # From [batch, heads, num_dest_tokens, embedding // heads] to [batch, num_dest_tokens, embedding]
        return merge_states(tf.transpose(x, [0, 2, 1, 3]))

    def mask_attn_weights(w, mask):
        # w shape [batch, heads, num_dest_tokens, num_src_tokens]
        _, _, nd, ns = shape_list(w)
        if len(shape_list(mask)) == 2:
            mask = tf.reshape(mask, [1, 1, nd, ns])
        else:
            # [batch, 1, num_dest_tokens, num_src_tokens]
            mask = tf.expand_dims(mask, axis=1)

        mask = tf.cast(mask, w.dtype)
        w = w*mask - tf.cast(1e10, w.dtype)*(1-mask)
        return w

    def multihead_attn(q, k, v, mask):
        # q has shape [batch, heads, num_dest_tokens, embedding // heads]
        # k, v have shape [batch, heads, num_src_tokens, embedding // heads]

        # shape [batch, heads, num_dest_tokens, num_src_tokens]
        w = tf.matmul(q, k, transpose_b=True)
        w = w * tf.compat.v1.rsqrt(tf.cast(v.shape[-1].value, w.dtype))

        w = mask_attn_weights(w, mask)
        w = tf.nn.softmax(w, name='attn_weights')

        # shape [batch, heads, num_dest_tokens, embedding // heads]
        a = tf.matmul(w, v)
        return a

    with tf.compat.v1.variable_scope(scope):
        c = conv1d(x, 'c_attn', n_state*3, cpu_or_train=cpu_or_train)

        # all have shape [batch, heads, num_dest_tokens, embedding // heads]
        q, k, v = map(split_heads, tf.split(c, 3, axis=2))

        # shape [batch, 2, heads, num_dest_tokens, embedding // heads]
        present = tf.stack([k, v], axis=1)
        if past is not None:
            pk, pv = tf.unstack(past, axis=1)  # matches call to stack above

            # we can just prepend the past keys and the past values to the
            # current ones and we will just compute (ignoring batching):
            # 1) a = q x [pk k] with a.shape = [num_dest_tokens, num_past_tokens + num_dest_tokens]
            # 2) mask should already account for past
            # 3) softmax(masked(a)) has shape [num_dest_tokens, num_past_tokens + num_dest_tokens]
            # 4) then we can just reconstruct context in terms of [pv, v] to get a result of shape [num_dest_tokens, embedding]
            k = tf.concat([pk, k], axis=-2)
            v = tf.concat([pv, v], axis=-2)
        a = multihead_attn(q, k, v, mask)
        a = merge_heads(a)
        a = conv1d(a, 'c_proj', n_state, cpu_or_train=cpu_or_train)
        return a, present


def mlp(x, scope, n_state, cpu_or_train=True):
    with tf.compat.v1.variable_scope(scope):
        nx = x.shape[-1].value
        h = gelu(conv1d(x, 'c_fc', n_state))
        h2 = conv1d(h, 'c_proj', nx, cpu_or_train=cpu_or_train)
        return h2


def transformer_block(x, scope, *, config: Config, mask: tf.Tensor, past=None, cpu_or_train=True):
    """
    Transforms x to h using the following operations:
      1) layer norm then multi-headed attention
      2) add (with skip connection from input)
      3) layer norm then multi layer perceptron
      4) add (with skip connection from output of 2)
    :param x: shape [batch, num_dest_tokens, embedding]
    :param past: see `attn` for details
    :param mask: see `attn` for details
    :return: (h,present) where
             - h.shape == x.shape
             - present -- see `attn` for details
    """
    with tf.compat.v1.variable_scope(scope):
        nx = x.shape[-1].value
        a, present = attn(layer_norm(x, 'ln_1'), 'attn', nx, past=past, config=config,
            mask=mask, cpu_or_train=cpu_or_train)
        x = x + a
        m = mlp(layer_norm(x, 'ln_2'), 'mlp', nx*4, cpu_or_train=cpu_or_train)
        x = x + m
        return x, present


def expand_tile(value, size):
    """Add a new axis of given size.
    :param value: shape [N1, N2, ...]
    :param size: scalar
    :return: shape [size, N1, N2, ...]
    """
    value = tf.convert_to_tensor(value, name='value')
    ndims = value.shape.ndims
    return tf.tile(tf.expand_dims(value, axis=0), [size] + [1]*ndims)


def positions_for(tokens, *, offset=None, pad_mask=None):
    """
    :param tokens: must be a 2d tensor, typically shape [batch, window]
    :param offset: scalar or rank 2 tensor ([batch, 1]) to offset the returned indices by this amount
    :param pad_mask: used for batched inputs, 0 if token is masked, 1 else. Shape [batch, window]
    :return: positions has shape tokens.shape, if pad_mask is None then
             positions[i,j] = offset + j, or
             positions[i,j] = offset[i, 0] + j
    """
    batch_size = tf.shape(tokens)[0]
    nsteps = tf.shape(tokens)[1]
    if offset is None:
        offset = 0
    positions = offset + expand_tile(tf.range(nsteps), batch_size)

    if pad_mask is None:
        return positions

    # if we have a pad_mask (i.e batched inputs), we need to shift positions
    # by the number of masked tokens. so, we count the number of masked tokens,
    # shift the positions, and then mask off the positions that aren't being used (otherwise
    # they would be negative, which doesn't work)
    n_ctx = shape_list(pad_mask)[1]
    pad_mask = tf.cast(pad_mask, tf.int32)
    # NOTE: use n_ctx - reduce_sum(mask) instead of reduce_sum(1-mask) as the sub op
    # is noticeable in profiles, esp when pad_mask is large & batched

    n_masked = n_ctx - tf.reduce_sum(pad_mask, axis=1, keepdims=True)
    return (positions - n_masked) * pad_mask


def tile_pasts(pasts: List[tf.Tensor], *, width: tf.Tensor) -> List[tf.Tensor]:
    """
    :param pasts: shape [layers, batch, 2, heads, window, embedding // heads]
    :return: shape [layers, batch*width, 2, heads, window, embedding // heads]

    Tile entries of pasts up to width, typically used as part of batched beam search
    where width is the width of the search. 
    
    In particular if the original context is [c1,c2] with shape [batch=2, window],
    Then we want to tile the context (and pasts) to [c1,c1,c2,c2] (assuming width = 2).

    """
    # TODO: use repeat once we upgrade tf, otherwise we recreate gather
    # indices for each past
    with tf.name_scope('build_batch_indices'):
        batch = shape_list(pasts[0])[0]
        # [batch, 1]
        batch_idxs = tf.expand_dims(tf.range(0, batch), axis=-1)
        # [batch, width]
        batch_idxs = tf.tile(batch_idxs, [1, width])
        # [batch*width]
        batch_idxs = tf.reshape(batch_idxs, [-1])

    tiled = []
    for layer, past in enumerate(pasts):
        # past has shape [batch, 2, heads, window, embedding // heads]
        # need to tile to [batch*width, 2, heads, window, embedding // heads]
        with tf.name_scope(f'tile_past_{layer}'):
            past = tf.gather(past, batch_idxs, name='past_tiled')
            tiled.append(past)
    return tiled


def add_past_mask(*, present_mask: tf.Tensor, valid_past_tokens: tf.Tensor) -> tf.Tensor:
    """
    :param present_mask: shape [batch, n_dst, n_dst]
    :param valid_past_tokens: shape [batch, n_past_tokens]
    :param return: [batch, n_dst, n_past_tokens + n_dst_tokens]
    """
    batch, n_dst = shape_list(present_mask)[:-1]

    # [batch, 1, n_past_tokens]
    past = tf.expand_dims(valid_past_tokens, axis=1)

    # [batch, n_dst, n_past_tokens]
    past = repeat(input=past, num=n_dst, axis=1)

    return tf.concat([past, present_mask], axis=-1)
