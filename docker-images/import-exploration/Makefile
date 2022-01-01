BUILD_DIR := /tmp/import-exploration-image

SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))

image:
	# Could also use setup.py sdist but we want to avoid getting the host python interpreter involved
	rm -f kite-python.tar
	tar cf $(SELF_DIR)/kite-python.tar \
		--exclude build \
		--exclude lib \
		--exclude dist \
		--exclude __pycache__ \
		--exclude '*.pyc' \
		--exclude kite-python.egg \
		-C $(SELF_DIR)/../.. \
		kite-python
	docker build -t kiteco/import-exploration $(SELF_DIR)

push: build
	docker push kiteco/import-exploration
