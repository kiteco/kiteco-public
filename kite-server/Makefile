.PHONY: deployment-token clean

deployment-token:
	openssl rand -hex 32

deployment-id:
	python -c 'import hashlib; import base64; print(base64.b64encode(hashlib.sha256(("${TOKEN}\n").encode()).digest()))'

clean:
	rm -rf kite-teams-ctl kite-server*

kite-teams-ctl:
	GOOS=linux go build ./tunable-models/cmds/kite-team-ctl


tag-all:
ifeq ($(TAG),)
	$(error Must set TAG to push)
endif
	cd edge-ingress && make TAG=$(TAG) build push
	cd metadata && make TAG=$(TAG) clean build push
	cd models-ingress && make TAG=$(TAG) build push
	cd models-stats-proxy && make TAG=$(TAG) clean build push
	cd tunable-models && make TAG=$(TAG) build push

VERSION = v0.6.0
DIR = kite-server-$(VERSION)

kite-server.tgz: clean kite-teams-ctl
	# setup deployment token
ifeq ($(TOKEN),)
	$(error Must set TOKEN to build kite-server.tgz)
endif

	mkdir $(DIR)
	# to store tuned models, mounted by tunable-models
	mkdir $(DIR)/tuned-models
	# to store code to tune on, mounted by tunable-models
	mkdir $(DIR)/repositories
	# to store kite-team-ctl binary
	mkdir $(DIR)/bin

	echo $(TOKEN) > $(DIR)/deployment_token
	@echo building with deployment_token $(TOKEN)

	cp kite-team-ctl $(DIR)/bin/
	cp nvidia-docker/configure_*.sh $(DIR)
	cp nvidia-docker/launch_kts.sh $(DIR)
	cp docker-stack.yml $(DIR)
	tar cvzf $@ $(DIR)
	rm -r $(DIR) kite-team-ctl
