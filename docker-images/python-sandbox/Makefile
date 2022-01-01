build: Dockerfile
	docker build -t kiteco/pythonsandbox .

push: build
	docker push kiteco/pythonsandbox

deploy: push
	ssh curation.kite.com docker pull kiteco/pythonsandbox
