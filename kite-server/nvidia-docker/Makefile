.PHONY: pypublish build-image

pypublish:
	rm -r dist
	python setup.py sdist
	twine upload dist/*

build-image:
	packer build -force packer.json

vagrant:
	packer build -only=vbox-nvidia-docker-puppet -force packer.json
	vagrant box add --name kite/nvidia-docker-puppet output-vbox-nvidia-docker-puppet/package.box
