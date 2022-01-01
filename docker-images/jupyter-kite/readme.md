# Jupyter + Kite Docker Image

## Contents of the image
- Jupyter Notebook Scientific Python Stack from https://github.com/jupyter/docker-stacks
- JupyterLab 2.2.x
- JupyterLab-Kite
- Kite

## Building the image

You'll need to download the file `kite-updater.sh` from S3 (which can be found in the bucket : `kite-downloads/linux/[latest version]`) into this directory.

Then to build the image you'll need to run from this folder:
```
docker build -t kiteco/jupyterhub-kite .
```

## Pushing to Docker Hub

After testing the image, you can tag it:
```
docker tag kiteco/jupyterhub-kite kiteco/jupyterhub-kite:[hex of the image]
```
> 🛑 Make sure you're authenticated for Docker Hub (with `docker login`)

Then push it into the Docker Hub repo with :
```
docker push kiteco/jupyterhub-kite:[image hex]
```
The tag id will be used in the Helm chart to specify which image to use. 

Our repository is located at https://hub.docker.com/r/kiteco/jupyterhub-kite.

### AWS
Building the image and pushing it to Docker Hub can be quite resource intensive. Our AWS EC2 instance "test-linux" can be used instead.

> 🛑 Make sure you have the [`kite-dev.pem`](https://github.com/kiteco/kiteco#ssh-access) certificate to `ssh` into the instance.

The AWS CLI and Docker is already installed and configured.

The [`kiteco`](https://github.com/kiteco/kiteco) repository should be present in the home directory. Perform a `git pull` to get the latest `Dockerfile`. 

> ⚠️ Don't forget to download `kite-updater.sh` from S3 into this directory.

## Running the image

```sh
sudo docker run 
            --dns=8.8.8.8 -it 
            -p 8888:8888 
            -v $PWD/notebooks:/home/jovyan/notebooks 
            -e KITE_USER=kite_account_email \
            -e KITE_PASSWORD=kite_account_password \
            kiteco/jupyterhub-kite
```

- The dns settings is a workaround for a bug making dns resolution really slow, which was making kite login to fail.
- Port 8888 is the port where jupyter lab is running
- Mounting an outside folder to /home/jovyan/notebooks allows notebooks to persist when the container is killed
- KITE_USER and KITE_EMAIL are used to login into Kite. If the env var is not defined, only Kite Free features will be available. 


To avoid putting credential in the command line, it is also possible to mount a credential file in `/usr/share/kite/kite-credentials`:

```sh
sudo docker run 
            --dns=8.8.8.8 -it 
            -p 8888:8888 
            -p 46624:46625 
            -v $PWD/notebooks:/home/jovyan/notebooks 
            -v $PWD/kite-credentials:/usr/share/kite/kite-credentials 
            kiteco/jupyterhub-kite
```

            
This file is then sourced in the container, so it should export all the required env variables:

```sh
export KITE_USER=email_for_the_kite_account
export KITE_PASSWORD=password_for_the_kite_account
```
