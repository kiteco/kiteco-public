all:

images/docker/image:
	cd images/docker && docker build -t kiteco/concourse .

images/docker/push: images/docker/image
	docker push kiteco/concourse

pipelines/bundle-plugins/set:
	fly -t kite sp -p bundle-plugins -c pipelines/bundle-plugins/pipeline.yml

YTT_ARGS=''
ifneq ($(BRANCH),)
YTT_ARGS="--data-value dev_branch=$(BRANCH)"
endif

BE_SVCS_DIR=pipelines/deploy-backend-services
$(BE_SVCS_DIR)/%/set: PIPELINE=$*
$(BE_SVCS_DIR)/%/set:
	ytt "$(YTT_ARGS)" -f $(BE_SVCS_DIR)/pipeline-template.ytt.yml -f $(BE_SVCS_DIR)/data-defaults.ytt.yml -f $(BE_SVCS_DIR)/$(PIPELINE)/data.ytt.yml > $(BE_SVCS_DIR)/$(PIPELINE)/pipeline.yml
	fly -t kite sp -l secrets.yml -p $(PIPELINE) -c $(BE_SVCS_DIR)/$(PIPELINE)/pipeline.yml
	rm $(BE_SVCS_DIR)/$(PIPELINE)/pipeline.yml

pipelines/%/set: PIPELINE=$*
pipelines/%/set:
	ytt -f pipelines/$(PIPELINE)/pipeline.ytt.yml > pipelines/$(PIPELINE)/pipeline.yml
	fly -t kite sp -p $(PIPELINE) -c pipelines/$(PIPELINE)/pipeline.yml
	rm pipelines/$(PIPELINE)/pipeline.yml
