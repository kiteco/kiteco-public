.PHONY: build run-local push

build: models/all-langs-large models/go-large models/js-large models/py-large
	go build github.com/kiteco/kiteco/local-pipelines/lexical/train/cmds/localtrain
	go build ./cmds/kite-team-server
	docker image build -t kiteco/kite-server-tunable-models -f Dockerfile ../..
	rm -f localtrain kite-team-server

push:
ifeq ($(TAG),)
	$(error Must set TAG to push)
endif
	docker tag kiteco/kite-server-tunable-models kiteco/kite-server-tunable-models:$(TAG)
	docker push kiteco/kite-server-tunable-models:$(TAG)

run-local:
	docker run --rm --gpus all \
		-p 8502:8502 \
		-p 8501:8501 \
    	-p 8500:8500 \
    	-v $(PWD)/repositories:/repositories \
    	-v ${PWD}/tuned-models:/tuned-models \
    	-t kiteco/kite-server-tunable-models

models/all-langs-large:
	mkdir -p models
	aws s3 sync s3://kite-data/run-db/2020-10-13T17:56:18Z_lexical-model-experiments/out_text__python-go-javascript-jsx-vue-css-html-less-typescript-tsx-java-scala-kotlin-c-cpp-objectivec-csharp-php-ruby-bash_lexical_context_512_embedding_720_layer_12_head_12_vocab_20000_steps_50000_batch_40 models/all-langs-large \
		--exclude="*" --include="tfserving*" --include="saved_model*" --include="config.json" --include="searchconfig.json" --include="ident-vocab-entries.bpe" --include="lexical_model.frozen.pb"

models/go-large:
	mkdir -p models
	aws s3 sync s3://kite-data/run-db/2020-06-23T01:08:16Z_tfserving-batched-v1/out_go_lexical_context_512_embedding_720_layer_12_head_12_vocab_10000_steps_4000000_batch_40 models/go-large \
		--exclude="*" --include="tfserving*" --include="saved_model*" --include="config.json" --include="searchconfig.json" --include="ident-vocab-entries.bpe" --include="lexical_model.frozen.pb"

models/py-large:
	mkdir -p models
	aws s3 sync s3://kite-data/run-db/2020-06-23T01:08:16Z_tfserving-batched-v1/out_python_lexical_context_512_embedding_720_layer_12_head_12_vocab_10000_steps_5000000_batch_50 models/py-large \
		--exclude="*" --include="tfserving*" --include="saved_model*" --include="config.json" --include="searchconfig.json" --include="ident-vocab-entries.bpe" --include="lexical_model.frozen.pb"

models/js-large:
	mkdir -p models
	aws s3 sync s3://kite-data/run-db/2020-06-23T01:08:16Z_tfserving-batched-v1/out_javascript_lexical_context_512_embedding_720_layer_12_head_12_vocab_10000_steps_5000000_batch_50 models/js-large \
		--exclude="*" --include="tfserving*" --include="saved_model*" --include="config.json" --include="searchconfig.json" --include="ident-vocab-entries.bpe" --include="lexical_model.frozen.pb"
