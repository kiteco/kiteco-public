image:
	GOOS=linux go build github.com/kiteco/kiteco/kite-go/cmds/local-code-worker
	docker build -t kiteco/local-code-worker .

push:
	docker push kiteco/local-code-worker

pull:
	docker pull kiteco/local-code-worker
	docker pull kiteco/process-docs

run:
	mkdir -p $(LOCAL_WORKER_STORAGE)
	touch $(SLOT_DB_URI)
	docker run -it \
		-e AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) \
		-e AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) \
		-e SLOT_DB_URI=/slots.db \
		-e SLOT_DB_DRIVER=$(SLOT_DB_DRIVER) \
		-e LOCAL_WORKER_STORAGE=/files \
		-e LOCAL_WORKER_REQUEST_QUEUE=$(LOCAL_WORKER_REQUEST_QUEUE) \
		-e LOCAL_WORKER_RESPONSE_QUEUE=$(LOCAL_WORKER_RESPONSE_QUEUE) \
		-e LOCAL_WORKER_BUCKET=$(LOCAL_WORKER_BUCKET) \
		-v $(SLOT_DB_URI):/slots.db \
		-v $(LOCAL_WORKER_STORAGE):/files \
		kiteco/local-code-worker
