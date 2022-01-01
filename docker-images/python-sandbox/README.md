This Dockerfile builds a basic python sandbox in which code examples will be run. Note that the code example authoring system does _not_ itself run inside a docker container. Rather, the authoring system starts docker containers and runs code examples inside them in order to isolate them from the rest of the system.

To build this docker image:

    make image

To deploy the image to the curation server, first make sure you are logged in (if you get "too many redirects" at this point, see below):

	docker login

You will find the credentials in the quip doc named "Credentials". Finally, to deploy the updated image to the curation server:

	make deploy

The above will push the image to a public Docker Hub repository and then pull that repository down to curation.kite.com, so take care not to put any non-public stuff into the image.

Appendix: Workaround for "too many redirects" during docker login. Run inside vagrant:

	sudo bash -c 'echo "nameserver 8.8.8.8" > /etc/resolv.conf'
	sudo service docker restart

Sorry for this awful hack! -Alex
