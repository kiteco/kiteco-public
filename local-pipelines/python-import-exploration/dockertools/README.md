# General
This binary exposes tools for building dockerfiles and dockerimages for import exploration.

# Dockerfiles

The `files` command builds a set of Dockerfiles for a set of packages.

## Tips
- Prefer `apt install` to `apt-get install`.
- Always prepend `apt install pkg` commands with `apt update &&` since we delete the package lists
  after each call to `apt install` to minimize image size.
- Always use the flags `-y --no-install-recommends` with `apt install` to ensure that we
  are not prompeted to confirm installation (`-y`) and to make sure we only install required dependencies
  with the package to minimize image size.

## Example usage
`./dockertools files --in packages --out dockerfiles/`
See `./dockertools help files` for details.

## Input Format
See `../internal/pkg/README.md` for the package file format.

### Examples

The following input
```
PACKAGE django 1.10

INSTALL pip install django==1.10

PACKAGE numpy 1.8
DEP apt install liblapack-dev
INSTALL apt install python-numpy=1.8
```
will produce 2 dockerfiles named `django__1.10` and `numpy__1.8` in the specified output directory.

## Output
- The output dockerfiles will be written to the provided directory.
- Each dockerfile will have a name of the form `PACKAGE__VERSION` in all lower case.

# Dockerimages

## Environment
- `docker-machine` must be installed for osx/windows, see: https://docs.docker.com/machine/install-machine/.
- If on osx/windows you must have a docker machine to build the images in, this can be accomplished via `docker-machine create default`
- You must be logged into hub.docker.com/kiteco, see https://docs.docker.com/engine/reference/commandline/login/
- If you get errors about needing permissions to run docker then run `sudo usermod -aG docker user-name` in the terminal to add your `user-name`
  to the docker group.

## Notes
- Unless a docker registry url is specified the image will not be uploaded and will only exist on this machine.
- `/entrypoint/entrypoint.py` is the entrypoint for the exploration, edit with care,
  any modidications to this file that affect all packages will require rebuilding all docker-images.
- If running on an azure instance you typically need to run all `docker` commands with `sudo`.

## buildimage
The `buildimage` command builds a single dockerimage for a dockerfile.

### Example usage
`./dockertools buildimage dockerfiles/abc --cert path/to/docker/certs`
- The name of the image will be extracted from the name of the input dockerfile, note that all names will be converted to lowercase
  because docker...
- See `./dockertools help buildimage` for details.

## buildimages
The `buildimages` command builds a set of dockerimages from a set of dockerfiles.

### Example usage
`./dockertools buildimages dockerfiles --cert path/to/docker/certs`
- `dockerfiles` is the path to a directory containing dockerfiles, the names of the images will be extracted from the
      names of the entries in the input directory (after being lowercased because ...docker).
      All entries in the input directory will be treated as
      dockerfiles.
- See `./dockertools help buildimages` for details.
- TIP: if running on a test instance, make sure to use the `nohup` command to ensure that the process is not killed if you log out,
  e.g: `nohup ./dockertools buildimages dockerfiles &> out.txt &`


## deleteimage
The `deleteimage` command deletes a single dockerimage.

### Example usage
`./dockertools deleteimage imagename --cert path/to/docker/certs`
- `imagename` is the name of the image to be deleted.

## deleteimages
The `deleteimages` command deletes a set of dockerimages.

### Example usage
`./dockertools deleteimages imagenames --cert path/to/docker/certs`
- `imagenames` is the path to a directory containing dockerfiles, the names of the images will be extracted from the
      names of the entries in the input directory (after being lowercased because ...docker).
      All entries in the input directory will be treated as
      dockerfiles.
- See `./dockertools help deleteimages` for details.

# baseimage
This directory contains the Dockerfile for the base docker image for import exploration.

# TODO
- Reduce size of docker images, right now the base image is ~460 MB, we could reduce this pretty dramatically
  by using the alpine docker images instead of a full ubuntu image. However this is a bit painful since we
  can only use the `apk` package registry which is much more limited than the `apt` registry. Alternatively we
  could modify the proccess so that pure python packages run in the alpine box and other packages
  that require more advanced dependency handling use the full linux box. We could also remove some of the
  dev dependencies (e.g build-essentials, pythonx-dev) but then we need to figure out which
  python libraries need these dependencies which is pretty painful.
- We should push the images to an internal registry since we cannot put them all on dockerhub.
- Add a git hash for pkgexploration to the dockerimage so we can mark the version. This is a bit annoying
  since the natural place to add the hash is in the dockerfile, but this does not actually guarantee
  that the image will be built with this version of kite.
- Sometimes when running the `buildimages` command with alot of images the following gets printed to stderr:
  `device or resource busy`, this seems to be an issue related to the `--force-rm` flag passed to `docker build`
  to remove the intermediate containers. The images all seem to build correctly however.
