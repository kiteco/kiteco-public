# General
Explore a package or packages.

# Package
Explore a specified package in a docker image.

## Usage
`./explore package --out out/dir DOCKERIMAGE`
- `DOCKERIMAGE` is the name of the docker image containing the installed
package for exploration.
- `out/dir` is the output directory to write logs and data to.
- The output graph will be written to `out/dir/DOCKERIMAGE.json`
- See `./explore help package` for more details.

# Packages
Explore a set of packages.

## Usage
`./explore packages --out out/dir ../dockertools/dockerfiles`
- `dockerimages` os the path to a directory containing dockerfiles whose names are the names of the dockerimages
  in which to run exploration.
- `out/dir` is the output directory to write logs and data to.
- The output graphs will be written to `out/dir/DOCKERIMAGE.json`, where `DOCKERIMAGE` is the
  name of one of the dockerimages that was explored.
- See `./explore help packages` for more details.
- TIP: use `nohup` when running on a remote instance to make sure that the process does not get killed
  if you get disconnected. e.g `nohup ./explore packages ../dockertools/dockerfils/ &> out.txt &`
- TIP: if you get errors about needing permissions to run docker then run `sudo usermod -aG docker user-name` in the terminal to add your `user-name`
  to the docker group.

# Known issues
- `django-haystack` -- needs preimport hooks since it tries to access django settings when it is imported.
- `django-pyscss` -- needs preimport hooks since it tries to access django settings when it is imported.
- `django-tables2` -- needs preimport hooks since it tries to access django settings when it is imported.
