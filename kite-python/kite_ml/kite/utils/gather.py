import tensorflow as tf


def gather_elems_2d(params: tf.Tensor, idxs: tf.Tensor, name: str, idxs_2d=False) -> tf.Tensor:
    """
    Gather elements from params specified by idxs. This is useful when dealing with params tensors
    that have shape [batch size, N] and we want to grab elements from
    each row of params as specified by idxs with shape [batch size, k].
    We typically cannot use tf.gather(params, idxs, axis=1)
    directly since we get a result that has shape [batch size, batch size, k]. Basically for each row of params
    gather will get all the entries from idxs.

    :param params: 2d tensor to gather elements from
    :param idxs: 1d or 2d tensor where idxs.shape[0] == params.shape[0]
    and idxs[i,j] is the column in row i of params that we want to gather the jth value from (see below)
    :param name: suffix to append to namespace and name of result
    :param idxs_2d: true if idxs has rank 2
    :return res: res.shape == idxs.shape and res[i,j] = params[i, idxs[i,j]]
    """

    with tf.name_scope('gather_elems_2d_'+name):
        if not idxs_2d:
            num_rows = tf.shape(params, name='num_rows')[0]

            # contents [[0], [1], ..., [num rows - 1]]
            # shape [num rows, 1]
            rows = tf.expand_dims(tf.range(num_rows), axis=1, name='rows')

            # shape [num rows, 1]
            idxs_expanded = tf.expand_dims(idxs, axis=1, name='idxs_expanded')

            # shape [num rows, 2]
            # contents [ [0, idxs[0]], [1, idxs[1]], ... [num rows - 1, idxs[num rows -1] ]
            rows_cols = tf.concat([rows, idxs_expanded], axis=1, name='rows_cols')
        else:
            shape = tf.shape(idxs, name='shape')
            num_rows = tf.cast(shape[0], idxs.dtype)
            num_cols = tf.cast(shape[1], idxs.dtype)

            # [num rows, 1]
            rows = tf.expand_dims(tf.range(num_rows, dtype=idxs.dtype), axis=1, name='rows')

            # shape [num rows, num cols]
            # [ [0,0, ... ,0], [1,1,...,1], ..., [num rows - 1, ..., num rows - 1] ]
            # so rows_tiled[i,j] = i
            rows_tiled = tf.tile(rows, [1, num_cols], name='rows_tiled')

            # [num rows, num cols, 1]
            rows_tiled_expanded = tf.expand_dims(rows_tiled, axis=2, name='rows_tiled_expanded')

            # [num rows, num cols, 1]
            idxs_expanded = tf.expand_dims(idxs, axis=2, name='idxs_expanded')

            # [num rows, num cols, 2]
            # rows_cols[i,j,0] = i, rows_cols[i,j,1] = idxs[i,j]
            rows_cols = tf.concat([rows_tiled_expanded, idxs_expanded], axis=2, name='rows_cols')

    # [idxs.shape]
    return tf.gather_nd(params, rows_cols, name=name)
