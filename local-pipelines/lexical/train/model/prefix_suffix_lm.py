from typing import Any, Dict, List, NamedTuple, Tuple

import time
import numpy as np
import tensorflow as tf

from kite.model.model import Model as BaseModel
from kite.utils.aggregator import SummaryInfo

from .config import Config, SearchConfig
from .feeder import Feed 
from .transformer import transformer_block, positions_for, layer_norm, lr_mask, tile_pasts, add_past_mask
from .search import search, SearchParams, SearchPlaceholders
from .util import shape_list, trim_padding, repeat, concat

class _Feed(NamedTuple):
    context_before: List[List[int]]
    context_after: List[List[int]]
    context_predict: List[List[int]]

    @classmethod
    def from_feed(cls, feed: Feed, n_vocab: int) -> '_Feed':
        # TODO: hacky
        # 50% context, 50% prediction slots
        delta = len(feed.context[0]) // 4
        before = []
        pred = []
        after = []
        for row in feed.context:
            if 4 * delta != len(row):
                row = row[:4*delta]
            before.append(row[:delta])
            pred.append(row[delta:3*delta])

            # TODO: hacky, sep token is always last token in vocab
            after_row = [n_vocab-1] + row[3*delta:]
            after.append(after_row)
        return _Feed(
            context_before=before,
            context_after=after,
            context_predict=pred,
        )

class _Placeholders(object):
    def __init__(self):
        with tf.name_scope('placeholders'):
            empty = np.empty([0,0], dtype=np.int64)
            # [batch, num_context_tokens_before]
            self.context_before: tf.Tensor = tf.compat.v1.placeholder_with_default(
                shape=[None, None], input=empty, name='context_before',  
            )

            # [batch, num_context_tokens_after]
            self.context_after: tf.Tensor = tf.compat.v1.placeholder_with_default(
                shape=[None, None], input=empty, name='context_after',
            )

            # [batch, num_context_tokens_predict]
            self.context_predict: tf.Tensor = tf.compat.v1.placeholder_with_default(
                shape=[None, None], input=empty, name='context_predict',  
            )
            
            # indicates if the model should just exit
            self.empty: tf.Tensor = tf.compat.v1.placeholder_with_default(
                shape=[], input=tf.constant(0, dtype=tf.int64), name='empty',
            )

    def dict(self) -> Dict[str, tf.Tensor]:
        return {
          self.context_before.name: self.context_before,
          self.context_after.name: self.context_after,
          self.context_predict.name: self.context_predict,
          self.empty.name: self.empty,
        }

    def feed_dict(self, feed: _Feed) -> Dict[tf.Tensor, Any]:
        return {
            self.context_before: feed.context_before,
            self.context_after: feed.context_after,
            self.context_predict: feed.context_predict,
        }

    def before(self, use_pad_mask=False):
        return _Ctx(self.context_before, use_pad_mask)
    
    def after(self, use_pad_mask=False):
        return _Ctx(self.context_after, use_pad_mask)
    
    def predict(self, use_pad_mask=False):
        return _Ctx(self.context_predict, use_pad_mask)

class _Ctx(object):
    def __init__(self, ctx, use_pad_mask=False):
        # [batch, num_tokens]
        self.ctx = ctx
        self.pad_mask = None
        if use_pad_mask:
             self.ctx, self.pad_mask, _ = trim_padding(ctx=ctx, mask=tf.cast(ctx > -1, tf.int64))
             # 0 out any negative ones in context, TODO: could send in mask like we do for other models
             self.ctx = self.ctx * self.pad_mask


class _PredBundle(NamedTuple):
    preds: tf.Tensor
    logits: tf.Tensor
    phs: _Placeholders = None


class _SearchBundle(NamedTuple):
    results: tf.Tensor
    probs: tf.Tensor
    search_phs: SearchPlaceholders = None


class Model(BaseModel):
    def __init__(self, config: Config, search_config: SearchConfig = SearchConfig(), training: bool=True, cpu: bool=True):
        self._start_time = time.time()
        self._config = config
        self._search_config = search_config
        self._cpu = cpu
        
        self._training = training
        self._phs = _Placeholders()
        
        self._pred_bundles: List[_PredBundle] = []
        self._search_bundle: _SearchBundle = None
        self._build()

    def _build(self):
        self._initialize_embeddings()

        if self._training:
            with tf.name_scope('train'):
                self._build_train_prediction_op()
                self._loss = self._build_loss_op()
                self._scalars = self._build_scalars()
                self._summaries = {k: v for k, v in self._scalars.items()}

        # TODO: empty string is global variable scope, this is a hack for backwards
        # compat to make sure we can reuse variables when we create the test graph
        if not self._training:
            with tf.compat.v1.variable_scope('', reuse=tf.compat.v1.AUTO_REUSE):
                if self._cpu:
                    with tf.name_scope('test'):
                        self._build_test_prediction_op()

                with tf.name_scope('search'):
                    self._build_search_op()


    def _initialize_embeddings(self):
        self._wpe_before = tf.compat.v1.get_variable(
            'wpe_before', [self._config.n_ctx, self._config.n_embd],
            initializer=tf.compat.v1.glorot_uniform_initializer(),
        )
        self._wpe_after = tf.compat.v1.get_variable(
            'wpe_after', [self._config.n_ctx, self._config.n_embd],
            initializer=tf.compat.v1.glorot_uniform_initializer(),
        )
        self._wpe_predict = tf.compat.v1.get_variable(
            'wpe_predict', [self._config.n_ctx, self._config.n_embd],
            initializer=tf.compat.v1.glorot_uniform_initializer(),
        )
        self._wte = tf.compat.v1.get_variable(
            'wte', [self._config.n_vocab, self._config.n_embd],
            initializer=tf.compat.v1.glorot_uniform_initializer(),
        )

    def _before_after_contexts(self, *, before: _Ctx, after: _Ctx):
        """
        :return: (embeddings, mask)
            - embeddings.shape == [batch, n_before + n_after, depth]
            - mask.shape == [batch, n_before + n_after, n_before + n_after]
        NOTE:
          - tokens in before/after have left -> right causality imposed
          - tokens in before (after) are not allowed to attend to tokens in after (before)
        """
        with tf.name_scope('embed_before'):
            before_pos = tf.gather(self._wpe_before, positions_for(before.ctx, pad_mask=before.pad_mask), name='pos')

            before_tok = tf.gather(self._wte, before.ctx, name='tok')
            before_embed = tf.identity(before_pos + before_tok, name='embed')
        
        with tf.name_scope('embed_after'):
            after_pos = tf.gather(self._wpe_after, positions_for(after.ctx, pad_mask=after.pad_mask), name='pos')

            after_tok = tf.gather(self._wte, after.ctx, name='tok')
            after_embed = tf.identity(after_pos + after_tok, name='embed')

        embeds = concat([before_embed, after_embed], axis=1, name='before_after_embed')
        
        # batch sizes can be different if before or after was not specified (this is for the test code path)
        batch_before, n_before = shape_list(before.ctx)
        batch_after, n_after = shape_list(after.ctx)

        with tf.name_scope('before_mask'):
            before_mask = lr_mask(n_before, n_before, tf.int64, batch=batch_before)
            if before.pad_mask is not None:
                pad_mask = tf.tile(tf.expand_dims(before.pad_mask, axis=1), [1, n_before, 1])
                before_mask = tf.cast(before_mask + pad_mask > 1, tf.int64)
            
            before_mask = concat([
              before_mask, 
              tf.zeros([batch_before, n_before, n_after], dtype=tf.int64),
            ], axis=-1)

        with tf.name_scope('after_mask'):
            after_mask = lr_mask(n_after, n_after, tf.int64, batch=batch_after)
            if after.pad_mask is not None:
                pad_mask = tf.tile(tf.expand_dims(after.pad_mask, axis=1), [1, n_after, 1])
                after_mask = tf.cast(after_mask + pad_mask > 1, tf.int64)
            
            after_mask = concat([
              tf.zeros([batch_after, n_after, n_before], dtype=tf.int64),
              after_mask,
            ], axis=-1)

        mask = concat([before_mask, after_mask], axis=1, name='before_after_mask')

        return embeds, mask

    def _predict_context(self, predict: _Ctx):
        """
        :return: (embeddings, mask)
                 - embeddings.shape == [batch, num_tokens_predict, depth]
                 - mask.shape == [batch, num_tokens_predict, num_tokens_predict]
        NOTE:
          - tokens in predict can attend to other tokens in predict but have left -> right causality imposed
        """
        with tf.name_scope('embed'):
            pos = tf.gather(self._wpe_predict, positions_for(predict.ctx, pad_mask=predict.pad_mask), name='pos')
            tok = tf.gather(self._wte, predict.ctx, name='tok')
            embed = tf.identity(pos + tok, name='embed')

        with tf.name_scope('mask'):
            batch, n_predict = shape_list(predict.ctx)
            mask = lr_mask(n_predict, n_predict, tf.int64, batch=batch)
            if predict.pad_mask is not None:
                pad_mask = tf.tile(tf.expand_dims(predict.pad_mask, axis=1), [1, n_predict, 1])
                mask = tf.cast(mask + pad_mask > 1, tf.int64)
        return embed, mask

    def _valid_tokens(self, mask):
        """
        :param mask: shape [batch, n_toks, n_toks]
        :return: shape [batch, n_toks]
        """
        # padded tokens aren't allowed to attend to themselves in the mask, so we can just check
        # which columns (per batch) of ba_mask have atleast 1 nonzero entry
        # [batch, n_toks]
        return tf.cast(tf.reduce_any(mask > 0, axis=1), mask.dtype)

    def _concat_contexts(self, *, ba, ba_mask, p, p_mask):
        """
        :param ba: shape [batch, n_before + n_after, depth]
        :param ba_mask: shape [batch, n_before + n_after, n_before + n_after]
        :param p: shape [batch, n_predict, depth]
        :param p_mask: shape [batch, n_predict, n_predict]
        :return: (embeddings, mask)
            - embeddings.shape == [batch, n_before + n_after + n_predict, depth]
            - mask.shape == [batch, n_before + n_after + n_predict, n_before + n_after + n_predict]
        NOTE:
          - tokens in before/after are not allowed to attend to tokens in predict
          - tokens in predict can attend to all tokens in before/after
        """
        embed = concat([ba, p], axis=1, name='embed')

        # batch sizes can be different if predict was not specified (this is for the test code path)        
        _, n_predict, _ = shape_list(p)
        batch_ba, n_ba, _ = shape_list(ba)

        # predict atttends to all tokens in ba except pad tokens 
        # [batch_ba, 1, n_ba]
        valid_ba = tf.expand_dims(self._valid_tokens(ba_mask), axis=1)
        p_mask = concat([
          tf.tile(valid_ba, [1, n_predict, 1]),
          p_mask,
        ], axis=-1, name='p_mask')

        ba_mask = concat([
          ba_mask,
          tf.zeros([batch_ba, n_ba, n_predict], dtype=p_mask.dtype),
        ], axis=-1, name='ba_mask')

        mask = concat([ba_mask, p_mask], axis=1, name='mask')

        return embed, mask

    def _transform(self, *, h: tf.Tensor, mask: tf.Tensor, pasts: List[tf.Tensor]=None):
        """
        :param pasts: pasts.shape == [n_layers, batch, None, None, num_past_tokens, None]
                      SEE: `transformer_block`
        :param h: state to transform, shape [batch, num_dest_tokens, depth]
        :param mask: shape [num_dest_tokens, num_src_tokens], should already account for pasts
        :return: (h_pred, presents)
                 - h_pred.shape == [batch, num_dest_tokens, depth]
                 - presents.shape == [n_layers, batch, None, None, num_dest_tokens, None]  
        """
        presents = []
        with tf.name_scope('transform'):
            for layer in range(self._config.n_layer):
                past = pasts[layer] if pasts else None
                h, present = transformer_block(h, f'h{layer}', past=past, mask=mask, config=self._config)
                presents.append(present)
            h = layer_norm(h, 'ln_f')
        return h, presents

    def _logits_and_preds(self, h: tf.Tensor, phs: _Placeholders) -> Tuple[tf.Tensor, tf.Tensor]:
        idx = -1
        if phs:
            # if there is no predict context then we need to grab the last before state
            idx = tf.cond(
              tf.equal(shape_list(phs.context_predict)[1], 0),
              true_fn=lambda: shape_list(phs.context_before)[1] - 1,
              false_fn=lambda: -1,
            )

        # [batch, depth]
        h_pred = tf.identity(h[:, idx, :], name='h_pred')

        # [batch, vocab]
        logits = tf.matmul(h_pred, self._wte, transpose_b=True, name='logits')
        preds = tf.nn.softmax(logits, axis=-1, name='preds')

        return logits, preds

    def _build_test_prediction_op(self):
        with tf.name_scope('encode'):
            ba, ba_mask = self._before_after_contexts(before=self._phs.before(), after=self._phs.after())
            p, p_mask = self._predict_context(self._phs.predict())
            ctx_h, ctx_mask = self._concat_contexts(ba=ba, ba_mask=ba_mask, p=p, p_mask=p_mask)

        with tf.name_scope('transform'):
            ctx_h, ctx_pasts = self._transform(h=ctx_h, mask=ctx_mask)

        with tf.name_scope('predict_init'):
            ctx_logits, ctx_preds = self._logits_and_preds(ctx_h, self._phs)
            pb = _PredBundle(phs=self._phs, preds=ctx_preds, logits=ctx_logits)
            self._pred_bundles.append(pb)

        with tf.name_scope('valid_past_tokens'):
            valid_past_tokens = self._valid_tokens(ctx_mask)

        def empty_pred():
            zeros = tf.zeros([0], dtype=tf.float32)
            return zeros, zeros

        def real_pred(phs: _Placeholders):
            with tf.name_scope('encode'):
                # [batch, num_pred_tokens, depth], [batch, num_pred_tokens, num_pred_tokens]
                h, mask = self._predict_context(phs.predict())

            batch = shape_list(h)[0]
            with tf.name_scope('tile_pasts'):
                # assumes initial batch was 1
                pasts = tile_pasts(ctx_pasts, width=batch)

            with tf.name_scope('build_mask'):
                valid_past_tokens_tiled = repeat(input=valid_past_tokens, num=batch)
                mask = add_past_mask(present_mask=mask, valid_past_tokens=valid_past_tokens_tiled)

            with tf.name_scope('transform'):
                h, _ = self._transform(h=h, mask=mask, pasts=pasts)

            with tf.name_scope('predict'):
                return self._logits_and_preds(h, None)

        for i in range(self._config.n_prediction_slots):
            with tf.name_scope(f'prediction{i}'):
                phs = _Placeholders()

                logits, preds = tf.cond(
                    tf.equal(phs.empty, 1),
                    true_fn=empty_pred,
                    false_fn=lambda: real_pred(phs),
                )
                logits = tf.identity(logits, name='logits')
                preds = tf.identity(preds, name='preds')


                pb = _PredBundle(phs=phs, preds=preds, logits=logits)
                self._pred_bundles.append(pb)

    def _build_search_op(self):
        with tf.name_scope('encode'):
            before, after = self._phs.before(use_pad_mask=True), self._phs.after(use_pad_mask=True)
            ctx_h, ctx_mask = self._before_after_contexts(before=before, after=after)

        with tf.name_scope('transform'):
            ctx_h, ctx_pasts = self._transform(h=ctx_h, mask=ctx_mask)
        
        with tf.name_scope('predict_init'):
            ctx_logits, ctx_preds = self._logits_and_preds(ctx_h, self._phs)

        sphs = SearchPlaceholders(defaults=self._search_config)
        with tf.name_scope('tile_for_search'):
            ctx_pasts = tile_pasts(ctx_pasts, width=sphs.width)
            valid_past_tokens = repeat(input=self._valid_tokens(ctx_mask), num=sphs.width)
        
        def next_logits(i: int, candidates: tf.Tensor) -> tf.Tensor:
            if i == -1:
                # TODO: kind of nasty, middle dim of 1 is for "search width"
                # [batch, 1, vocab]
                return tf.expand_dims(ctx_logits, axis=1)

            batch, width, depth = shape_list(candidates)
            with tf.name_scope('encode'):
                candidates_flat = tf.reshape(candidates, [batch*width, depth])

                # [batch*width, depth], [batch*width, depth, depth]
                h, mask = self._predict_context(_Ctx(candidates_flat))

            with tf.name_scope('build_mask'):
                mask = add_past_mask(present_mask=mask, valid_past_tokens=valid_past_tokens)
            
            with tf.name_scope('transform'):
                h, _ = self._transform(h=h, mask=mask, pasts=ctx_pasts)

            with tf.name_scope('predict'):
                # [batch*width, vocab]
                logits, _ = self._logits_and_preds(h=h, phs=None)
                logits = tf.reshape(logits, [batch, width, -1])
      
            return logits

        with tf.name_scope('search'):
            batch = shape_list(ctx_h)[0]
            sp = SearchParams(phs=sphs, depth=5, n_vocab=self._config.n_vocab, batch=batch, next_logits=next_logits)

            sr = search(sp)
            results = tf.identity(sr.results, name='results')
            probs = tf.identity(sr.probs, name='probs')
            self._search_bundle = _SearchBundle(search_phs=sphs, results=results, probs=probs)

    def _build_train_prediction_op(self):
        with tf.name_scope('encode'):
            before, after, predict = self._phs.before(), self._phs.after(), self._phs.predict()
            ba, ba_mask = self._before_after_contexts(before=before, after=after)
            p, p_mask = self._predict_context(predict)
            h, mask = self._concat_contexts(ba=ba, ba_mask=ba_mask, p=p, p_mask=p_mask)

        with tf.name_scope('transform'):
            h, _ = self._transform(h=h, mask=mask)

        with tf.name_scope('predict'):
            batch, n_ctx, depth = shape_list(h)

            h_flat = tf.reshape(h, [batch*n_ctx, depth], name='h_flat')
            logits_flat = tf.matmul(h_flat, self._wte, transpose_b=True, name='logits_flat')

            self._train_logits = tf.reshape(logits_flat, [batch, n_ctx, self._config.n_vocab], name='logits')
            self._train_preds = tf.nn.softmax(self._train_logits, name='preds')

    def _build_loss_op(self) -> tf.Tensor:
        with tf.name_scope('loss'):
            n_before = shape_list(self._phs.context_before)[1]
            n_after = shape_list(self._phs.context_after)[1]
            n_predict = shape_list(self._phs.context_predict)[1]

            # given the <n tokens in before, predict nth token
            loss_before = tf.nn.sparse_softmax_cross_entropy_with_logits(
              labels=self._phs.context_before[:, 1:], logits=self._train_logits[:, :n_before-1], name='before_vec', 
            )
            loss_before = tf.reduce_mean(loss_before, name='before')

            # given the <n tokens in after, predict nth token
            # NOTE: we don't measure loss on the sep token
            loss_after = tf.nn.sparse_softmax_cross_entropy_with_logits(
              labels=self._phs.context_after[:, 2:], logits=self._train_logits[:, n_before+1:n_before+n_after-1], name='after_vec',
            )
            loss_after = tf.reduce_mean(loss_after, name='after')

            # given the before and after tokens, and <n tokens in predict, predict nth token
            loss_predict = tf.nn.sparse_softmax_cross_entropy_with_logits(
              labels=self._phs.context_predict[:, 1:], logits=self._train_logits[:, -n_predict:-1], name='predict_vec',
            )
            loss_predict = tf.reduce_mean(loss_predict, name='predict')

            loss = (loss_before + loss_after + loss_predict) / 3.

            return tf.identity(loss, name='loss')

    def _build_scalars(self) -> Dict[str, tf.Tensor]:
        with tf.name_scope('scalars'):
            n_before = shape_list(self._phs.context_before)[1]
            n_after = shape_list(self._phs.context_after)[1]
            n_predict = shape_list(self._phs.context_predict)[1]

            predicted = tf.argmax(self._train_preds, axis=-1)

            before = tf.cast(tf.equal(predicted[:, :n_before-1], self._phs.context_before[:, 1:]), tf.float32)
            before = tf.reduce_mean(before)

            after = tf.cast(tf.equal(predicted[:, n_before+1:n_before+n_after-1], self._phs.context_after[:, 2:]), tf.float32)
            after = tf.reduce_mean(after)

            predict = tf.cast(tf.equal(predicted[:, -n_predict:-1], self._phs.context_predict[:, 1:]), tf.float32)
            predict = tf.reduce_mean(predict)

            accuracy = (before + after + predict) / 3.

            self._elapsed_time = tf.compat.v1.placeholder(dtype=tf.float32, shape=[], name='elapsed_time')
            return {
                'accuracy': accuracy,
                'elapsed_time': self._elapsed_time,
            }

    def loss(self) -> tf.Tensor:
        return self._loss

    def summary_infos(self) -> List[SummaryInfo]:
        return [SummaryInfo(k) for k in self._scalars]

    def summaries_to_fetch(self) -> Dict[str, tf.Tensor]:
        return self._summaries

    def feed_dict(self, feed: Feed, train: bool) -> Dict[tf.Tensor, Any]:
        feed_dict = self._phs.feed_dict(_Feed.from_feed(feed, self._config.n_vocab))
        feed_dict.update({self._elapsed_time: float(time.time()-self._start_time)})
        return feed_dict

    def placeholders_dict(self) -> Dict[str, tf.Tensor]:
        d = self._phs.dict()
        for p in self._pred_bundles:
            d.update(p.phs.dict())
        if self._search_bundle:
            d.update(self._search_bundle.search_phs.dict())
        return d

    def outputs_dict(self) -> Dict[str, tf.Tensor]:
        d = dict()
        if self._training:
            # because outputs cannot be empty...
            d[self._train_logits.name] = self._train_logits
            d[self._train_preds.name] = self._train_preds
        for p in self._pred_bundles:
            d[p.logits.name] = p.logits
            d[p.preds.name] = p.preds
        if self._search_bundle:
            d[self._search_bundle.results.name] = self._search_bundle.results
            d[self._search_bundle.probs.name] = self._search_bundle.probs
        return d

    def tfserving_inputs_dict(self) -> Dict[str, tf.Tensor]:
        inputs = {
            'context_before': self._phs.context_before,
            'context_after': self._phs.context_after,
        }
        return inputs

    def tfserving_outputs_dict(self) -> Dict[str, tf.Tensor]:
        return {
            # Beam fetches
            'search_results': self._search_bundle.results,
            'search_probs': self._search_bundle.probs
        }
