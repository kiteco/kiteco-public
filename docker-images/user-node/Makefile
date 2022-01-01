user-node:
	./build-user-node.sh

image: user-node
	docker build -t kiteco/user-node .

run: image
	docker run -it --rm \
		-e AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID) \
		-e AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY) \
		kiteco/user-node
