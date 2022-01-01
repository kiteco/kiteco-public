from typing import Any, Dict, List, Tuple, NamedTuple

import time
import tensorflow as tf

from kite.model.model import Model as BaseModel
from kite.utils.aggregator import SummaryInfo, AggregateOp

from .config import Config, SearchConfig
from .feeder import Feed
from .transformer import transformer_block, layer_norm, positions_for, tile_pasts, add_past_mask, lr_mask
from .search import search, SearchParams, SearchPlaceholders
from .util import shape_list, trim_padding, repeat


class _Placeholders(object):
    def __init__(self):
        with tf.name_scope('placeholders'):
            # shape [batch, num_context_tokens]
            self.context: tf.Tensor = tf.compat.v1.placeholder(
                dtype=tf.int64, shape=[None, None], name='context',
            )
            # shape [batch]
            self.langs: tf.Tensor = tf.compat.v1.placeholder(
                dtype=tf.int64, shape=[None], name='langs',
            )
            # shape [batch, num_context_tokens]
            # leading 0's for tokens to mask off, 1's for "on" tokens
            self.context_mask: tf.Tensor = tf.compat.v1.placeholder(
                dtype=tf.int64, shape=[None, None], name='context_mask',
            )

    def dict(self) -> Dict[str, tf.Tensor]:
        return {
            self.context.name: self.context,
            self.langs.name: self.langs,
            self.context_mask.name: self.context_mask
        }

    def feed_dict(self, feed: Feed) -> Dict[tf.Tensor, Any]:
        return {
            self.context: feed.context,
            self.langs: feed.langs,
            # context_mask not needed here, simply use
            # tf.ones_like(context) during training and predict ops
        }
    
    def n_ctx(self) -> tf.Tensor:
        return shape_list(self.context)[1]
    
    def n_batch(self) -> tf.Tensor:
        return shape_list(self.context)[0]


class _PredictionBundle(NamedTuple):
    preds: tf.Tensor
    logits: tf.Tensor
    phs: _Placeholders = None


class _SearchBundle(NamedTuple):
    results: tf.Tensor
    probs: tf.Tensor
    phs: SearchPlaceholders = None


def tfprint(tensor: tf.Tensor, name='') -> tf.Tensor:
    if name == '':
        name = tensor.name
    return tf.Print(tensor, [tensor], message="\n"+name+": ", summarize=10000)


class LexicalModel(BaseModel):
    def __init__(self, config: Config, search_config: SearchConfig = SearchConfig(),
                 training: bool = True, cpu: bool = True):

        self._start_time = time.time()
        self._config = config
        self._search_config = search_config
        self._placeholders = _Placeholders()

        self._cpu = cpu
        self._training = training
        self._prediction_ops = []
        self._search_bundle = None

        self._build()

    def _build(self):
        self._initialize_embeddings()

        # TODO: move these under the train name scope once we transition to only the new model
        self._build_train_prediction_op()
        self._loss = self._build_loss_op()
        self._scalars = self._build_scalars()
        self._summaries = {k: v for k, v in self._scalars.items()}

        # TODO: empty string is global variable scope, this is a hack for backwards
        # compat to make sure we can reuse variables when we create the test graph
        if not self._training:
            with tf.compat.v1.variable_scope('', reuse=True):
                if self._cpu:
                    with tf.name_scope('test'):
                        self._build_test_prediction_op()

                if not self._config.down_project_embds():
                    with tf.name_scope('search'):
                        self._build_search_prediction_op()

    def _initialize_embeddings(self):
        self._langs = None
        full_dim = self._config.n_embd
        if self._config.down_project_embds():
            full_dim = self._config.n_full_embd
            self._langs = tf.compat.v1.get_variable(
                'langs', [self._config.n_langs, full_dim, self._config.n_embd],
                initializer=tf.compat.v1.glorot_uniform_initializer(),
            )
        
        self._wpe = tf.compat.v1.get_variable(
            'wpe', [self._config.n_ctx, full_dim],
            initializer=tf.compat.v1.glorot_uniform_initializer())
        self._wte = tf.compat.v1.get_variable(
            'wte', [self._config.n_vocab, full_dim],
            initializer=tf.compat.v1.glorot_uniform_initializer())

    def _wte_wpe(self, lang):
        if not self._config.down_project_embds():
            return self._wte, self._wpe
        # [full_embd, n_embd]
        proj = self._langs[lang, :, :]
        # [n_vocab, n_embd] = [n_vocab, full_embd] x [full_embd, n_embd]
        wte = tf.matmul(self._wte, proj)
        # [n_ctx, n_embd] = [n_ctx, full_emb] x [full_embd, n_embd]
        wpe = tf.matmul(self._wpe, proj)
        return wte, wpe

    def _initial_embeddings(self, ids, *, wte, wpe, offset=0, context_mask=None):
        """
        return the initial embeddings for the provided token ids
        :param ids: shape [batch, num context tokens]
        :param offset: optional offset for the positional embedding ids, shape []
        :return: shape ids.shape + [embedding depth]
        """
        with tf.name_scope('initial_embeddings'):
            positions = positions_for(ids, offset=offset, pad_mask=context_mask)

            positional = tf.gather(wpe, positions, name='positional')
            tokens = tf.gather(wte, ids, name='tokens')

            # h.shape = ids.shape + [n_embd]
            h = tokens + positional
            return tf.identity(h, name='initial_embeddings')

    def _transform(self, *, h: tf.Tensor, mask: tf.Tensor, pasts: List[tf.Tensor]=None) -> Tuple[tf.Tensor, List[tf.Tensor]]:
        """
        Apply transformer stack of n layers to h to produce h_new
        :param h: initial input, has shape [batch, context, embedding]
        :param pasts: lists of pasts from a previous call, see block above for information on each element
        :return: (h_new, presents) where
                 - h_new.shape == h.shape
                 - len(presents) == n_layers, see transformer_block for information on each entry
        """
        cpu_or_train = self._cpu or self._training
        presents = []
        pasts = pasts if pasts else [None]*self._config.n_layer
        assert len(pasts) == self._config.n_layer

        with tf.name_scope('transformer'):
            for layer, past in enumerate(pasts):
                h, present = transformer_block(h, f'h{layer}', past=past, 
                    config=self._config, mask=mask, cpu_or_train=cpu_or_train)
                presents.append(present)
            
            h = layer_norm(h, 'ln_f')
        
        return h, presents

    def _prediction_logits(self, *, last_h, wte) -> tf.Tensor:
        """
        Logits are just the inner product between the last state of the last context token
        and the rows of the token embedding matrix
        """
        return tf.matmul(last_h, wte, transpose_b=True, name='prediction_logits')

    def _prediction(self, *, last_h, wte, phs: _Placeholders, is_empty=tf.constant(False)) -> _PredictionBundle:
        """
        Prediction is just
         - inner product last_h with the rows of the token embedding matrix
         - softmax
        :param last_h: [batch, embedding depth] the final hidden state of the last token
        :param is_empty: boolean scalar tensor indicating that we should not do any work
        :return: logits, predictions with shape [batch, vocab]
        """

        def empty_logits():
            batch_vocab = [phs.n_batch(), self._config.n_vocab]
            return tf.zeros(batch_vocab, dtype=last_h.dtype, name='empty_logits')

        def real_logits():
            return tf.identity(self._prediction_logits(last_h=last_h, wte=wte), name='real_logits')

        # [batch, vocab]
        logits = tf.cond(
            is_empty,
            true_fn=empty_logits,
            false_fn=real_logits,
            name='logits_cond',
        )

        logits = tf.identity(logits, name='logits')
        preds = tf.nn.softmax(logits, axis=-1, name='preds')
        return _PredictionBundle(phs=phs, logits=logits, preds=preds)

    def _build_test_prediction_op(self):
        with tf.name_scope('embed_context'):
            wte, wpe = self._wte_wpe(self._placeholders.langs[0])
            context = self._initial_embeddings(self._placeholders.context, wte=wte, wpe=wpe)

        with tf.name_scope('encode_context'):
            n_ctx = shape_list(context)[1]
            context_mask = tf.expand_dims(lr_mask(n_ctx, n_ctx, context.dtype), axis=0)
            context_final, context_states = self._transform(h=context, mask=context_mask)

        with tf.name_scope('predict_context'):
            self._first_pred_context = self._prediction(
                last_h=tf.identity(context_final[:, -1, :], name='last_h'),
                wte=wte, phs=self._placeholders,
            )

        with tf.name_scope('new_tokens_offset'):
            # use the length of the existing context as an offset
            # to ensure that the new tokens get new positional embeddings below
            offset = shape_list(self._placeholders.context)[1]

        # build prediction ops
        # because a placeholder (or node) may only be fed (fetched) once we have to
        # create multiple nodes, one for each prediction step.
        for i in range(self._config.n_prediction_slots):
            with tf.name_scope(f'prediction{i}'):
                # placeholders for new tokens
                phs = _Placeholders()

                def real_last_h():
                    # shape [batch, num new tokens, embedding depth]
                    h = self._initial_embeddings(phs.context, wte=wte, wpe=wpe, offset=offset)

                    # next few lines assume context batch size is 1
                    pasts = tile_pasts(context_states, width=phs.n_batch())

                    mask = lr_mask(phs.n_ctx(), phs.n_ctx(), h.dtype, batch=1)
                    mask = add_past_mask(present_mask=mask, valid_past_tokens=context_mask[:, -1, :])

                    # shape [batch, num new tokens, embedding depth]
                    h, _ = self._transform(h=h, pasts=pasts, mask=mask)
                    return tf.identity(h[:, -1, :], name='real_last_h')

                def empty_last_h():
                    shape = [shape_list(phs.context)[0], self._config.n_embd]
                    return tf.zeros(shape, dtype=context_final.dtype, name='empty_last_h')

                # handle case where client sends in -1 in context
                is_empty = tf.reduce_any(phs.context < tf.constant(0, dtype=phs.context.dtype))

                # shape [batch, embedding depth]
                last_h = tf.cond(
                    is_empty,
                    true_fn=empty_last_h,
                    false_fn=real_last_h,
                    name='h_cond',
                )

                # predict
                pred = self._prediction(last_h=last_h, wte=wte, phs=phs, is_empty=is_empty)

                self._prediction_ops.append(pred)

    def _build_search_prediction_op(self):
        # TODO: maybe add support for downprojecting embeddings for large models?
        wte, wpe = self._wte_wpe(-1)
        # remove excess padding to reduce the context size going into the transformer
        with tf.name_scope('trim_padding'):
            # [batch, n_ctx], [batch, n_ctx]
            context, pad_mask, num_padding = trim_padding(
                ctx=self._placeholders.context,
                mask=self._placeholders.context_mask, 
            )

        with tf.name_scope('embed_context'):
            context = self._initial_embeddings(context, wte=wte, wpe=wpe, context_mask=pad_mask)

        with tf.name_scope('build_mask'):
            n_ctx = shape_list(context)[1]

            # add padding mask and left right mask to find "and" of the two
            # [batch, 1, n_ctx]
            pad_mask_expanded = tf.expand_dims(pad_mask, axis=1)

            # [1, n_ctx, n_ctx]
            context_lr = lr_mask(n_ctx, n_ctx, pad_mask.dtype, batch=1)

            # [batch, n_ctx, n_ctx]
            context_mask = (context_lr + pad_mask_expanded) > 1

        with tf.name_scope('encode_context'):
            context_final, context_states = self._transform(h=context, mask=context_mask)

        with tf.name_scope('new_tokens_offset'):
            # use the length of the existing context as an offset
            # to ensure that the new tokens get new positional embeddings below
            offset = shape_list(pad_mask)[1]
            offset = offset - num_padding

        def next_logits(i: int, candidates: tf.Tensor) -> tf.Tensor:
            if i == -1:
                # [batch, vocab]
                logits = self._prediction_logits(last_h=context_final[:, -1, :], wte=wte)
                # [batch, 1, -1] -> middle 1 dim is for beam width. just 1 during init
                # TODO: this is kind of nasty
                logits = tf.expand_dims(logits, axis=1)
            else:
                # depth is really current_search_depth+1 since +1 comes from init
                batch, width, depth = shape_list(candidates)

                with tf.name_scope('build_mask'):
                    # [batch*width, depth, depth]
                    mask = lr_mask(depth, depth, context_mask_last_row.dtype, batch=batch*width)

                    # [batch*width, depth, n_ctx + depth]
                    mask = add_past_mask(present_mask=mask, valid_past_tokens=context_mask_last_row)


                # flattened to [b1_h1, b1_h2, b2_h1, b2_h2, ...]
                candidates_flattened = tf.reshape(candidates, [batch*width, depth])
                # [batch*width, window+depth, embedding depth]
                h, _ = self._transform(
                    h=self._initial_embeddings(candidates_flattened, offset=offset, wte=wte, wpe=wpe),
                    pasts=context_states, mask=mask,
                )
                logits = self._prediction_logits(last_h=h[:, -1, :], wte=wte)
                logits = tf.reshape(logits, [batch, width, -1])

            return logits

        with tf.name_scope('search'):
            phs = SearchPlaceholders(defaults=self._search_config)

            batch = shape_list(context_final)[0]
            sp = SearchParams(phs=phs, depth=5, n_vocab=self._config.n_vocab, batch=batch, next_logits=next_logits)

            # TODO: nasty, we do this tiling here so we can declare the placeholders
            # inside the `search/search` name scope for backwards compatibility
            with tf.name_scope('tile_for_search'):
                # tile stuff to match width of search
                # everything becomes batch -> batch*width
                context_states = tile_pasts(context_states, width=phs.width)
                offset = repeat(input=offset, num=phs.width)
                context_mask_last_row = repeat(input=context_mask[:, -1, :], num=phs.width)

            sr = search(sp)
            results = tf.identity(sr.results, name='results')
            probs = tf.identity(sr.probs, name='probs')
            self._search_bundle = _SearchBundle(phs=phs, results=results, probs=probs)

    def _build_train_prediction_op(self):
        cpu_or_train = (self._cpu or self._training)

        with tf.name_scope('encode'):
            wte, wpe = self._wte, self._wpe
            batch_dims = 0
            if self._config.down_project_embds():
                batch_dims = 1
                # self._langs.shape = [n_langs, full_embd, n_embd]
                # self._placeholders.langs.shape = [batch], where each elem is in 0, ..., n_langs - 1
                # [batch, full_embd, n_embd]
                langs = tf.gather(self._langs, self._placeholders.langs)
                
                # [batch, vocab, n_embd] = [vocab, full_embd] x [batch, full_embd, n_embd]
                wte = tf.matmul(self._wte, langs)

                # [batch, n_ctx, n_embd] = [n_ctx, full_embd] x [batch, full_embd, n_embd]
                wpe = tf.matmul(self._wpe, langs)

            # self._placeholders.context.shape = [batch, n_ctx]
            # [batch, n_ctx, n_embd]
            tokens = tf.gather(wte, self._placeholders.context, batch_dims=batch_dims)
            pos = tf.gather(wpe, positions_for(self._placeholders.context), batch_dims=batch_dims)
            h = tokens + pos

        with tf.name_scope('transformer'):
            batch, n_ctx = shape_list(self._placeholders.context)
            
            mask = lr_mask(n_ctx, n_ctx, h.dtype)
            for layer in range(self._config.n_layer):
                h, _ = transformer_block(h, 'h%d' % layer, config=self._config, mask=mask, cpu_or_train=cpu_or_train)
            # [batch, n_ctx, n_embd]
            h = layer_norm(h, 'ln_f')

            # shape [batch, n_ctx, vocab] = [batch, n_ctx, n_embd] x [batch, n_embd, vocab]
            self._logits = tf.matmul(h, wte, transpose_b=True, name='logits')

            # [batch, batch, vocab] = [batch, n_embd] x [batch, vocab, n_embd]
            last_logits = tf.matmul(h[:, -1, :], wte, transpose_b=True)

        with tf.name_scope('prediction'):
            self._pred = tf.nn.softmax(self._logits, axis=-1, name='pred')
            self._last = tf.nn.softmax(last_logits, axis=-1, name='last')
            self._last_logits = tf.identity(last_logits, name='last_logits')

    def _build_loss_op(self) -> tf.Tensor:
        with tf.name_scope('loss'):
            # Loss: given <n tokens, predict n token
            losses = tf.nn.sparse_softmax_cross_entropy_with_logits(
                labels=self._placeholders.context[:, 1:], logits=self._logits[:, :-1], name='loss_vec')
            return tf.reduce_mean(losses, name='loss')

    def _build_metrics_op(self):
        with tf.name_scope('metrics'):
            # Accuracy
            predicted = tf.argmax(self._pred, axis=-1)
            equal_first = tf.cast(tf.equal(predicted[:, :-1], self._placeholders.context[:, 1:]), tf.float32)
            self._accuracy = tf.reduce_mean(equal_first)
            self._elapsed_time = tf.compat.v1.placeholder(dtype=tf.float32, shape=[], name='elapsed_time')

    def _build_scalars(self) -> Dict[str, tf.Tensor]:
        with tf.name_scope('scalars'):
            self._build_metrics_op()
            return {
                'accuracy': self._accuracy,
                'elapsed_time': self._elapsed_time,
            }

    def placeholders_dict(self) -> Dict[str, tf.Tensor]:
        d = self._placeholders.dict()
        for p in self._prediction_ops:
            d.update(p.phs.dict())
        if self._search_bundle:
            d.update(self._search_bundle.phs.dict())
        return d

    def feed_dict(self, feed: Feed, train: bool) -> Dict[tf.Tensor, Any]:
        feed_dict = self._placeholders.feed_dict(feed)
        feed_dict.update({self._elapsed_time: float(time.time()-self._start_time)})
        return feed_dict

    def outputs_dict(self) -> Dict[str, tf.Tensor]:
        d = {
            self._last.name: self._last,
            self._last_logits.name: self._last_logits,
        }
        if not self._training and self._cpu:
            d.update({
                self._first_pred_context.logits.name: self._first_pred_context.logits,
                self._first_pred_context.preds.name: self._first_pred_context.preds,
            })
        for p in self._prediction_ops:
            d[p.logits.name] = p.logits
            d[p.preds.name] = p.preds
        if self._search_bundle:
            d[self._search_bundle.results.name] = self._search_bundle.results
            d[self._search_bundle.probs.name] = self._search_bundle.probs
        return d

    def loss(self) -> tf.Tensor:
        return self._loss

    def summary_infos(self) -> List[SummaryInfo]:
        return [SummaryInfo(k) for k in self._scalars]

    def summaries_to_fetch(self) -> Dict[str, tf.Tensor]:
        return self._summaries

    def tfserving_inputs_dict(self) -> Dict[str, tf.Tensor]:
        inputs = {
            'context': self._placeholders.context,
            'context_mask': self._placeholders.context_mask,
            'valid_prefix_ids': self._search_bundle.phs.valid_prefix_ids,
        }
        return inputs

    def tfserving_outputs_dict(self) -> Dict[str, tf.Tensor]:
        return {
            # Beam fetches
            'search_results': self._search_bundle.results,
            'search_probs': self._search_bundle.probs
        }


def scatter_update_3d(tensor, idxs, updates, name=''):
    """
    tensor: [batch, rows, 1]
    idxs:   [batch, row_idx (N)]
    returns [batch, rows, 1] with updates applied according to row_idx above
    """

    batch = shape_list(idxs)[0]
    num_rows = shape_list(idxs)[1]

    # [batch]
    rows = tf.cast(tf.range(batch), idxs.dtype)

    # [batch] -> [batch, 1, 1]
    for dim in range(2):
        rows = tf.expand_dims(rows, axis=1)

    # [batch, num_rows, 1]
    rows = tf.tile(rows, [1, num_rows, 1])

    # [batch, num_rows, 1] + [batch, num_rows, 1] to create
    # an index (row, col) into tensor on the first dimension
    row_cols = tf.concat([rows, idxs], axis=2)

    return tf.compat.v1.tensor_scatter_update(
        tensor, row_cols, updates, name=name)
