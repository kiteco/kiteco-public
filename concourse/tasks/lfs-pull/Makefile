.PHONY: all

docker:
	docker build -t kiteco/concourse.lfs-pull .

docker-push: docker
	docker push kiteco/concourse.lfs-pull
