build: Dockerfile
	docker build -t kiteco/build-frontend .

shell: build
	docker run -v $(PWD):/kiteco -i -t kiteco/build-frontend bash

push: build
	docker push kiteco/build-frontend

deploy: push
	ssh build.kite.com docker pull kiteco/build-frontend
