build: Dockerfile
	docker build -t kiteco/bashsandbox .

push: build
	docker push kiteco/bashsandbox

deploy: push
	ssh curation.kite.com docker pull kiteco/bashsandbox
