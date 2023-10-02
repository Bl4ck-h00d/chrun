#!/bin/bash

# script to get the linux file system (for dynamic linking) from the docker container

set -e

defaultImage="hello-world"

image="${1:-$defaultImage}"
container=$(docker create "$image") # spin up a docker container

docker export "$container" -o "./assets/${image}.tar.gz" > /dev/null # get the filesystem as tar file from the container
docker rm "$container" > /dev/null # delete the container

docker inspect -f '{{.Config.Cmd}}' "$image:latest" | tr -d '[]\n' > "./assets/${image}-cmd" # get the default cmd

echo "Image content stored in assets/${image}.tar.gz"