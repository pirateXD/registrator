#!/bin/sh
git pull && echo "pull registrator"

registryHost=$(grep "^registry" docker_run.config | awk -F'=' '{print $2}')
tag=$(grep "^version" docker_run.config | awk -F'=' '{print $2}')
if [[ -z "$tag" ]]; then
    tag="latest"
fi
echo "tag is: $tag"
# export GOPATH=$(cd ../../..; pwd)
echo "GOPATH : $GOPATH"
export GOOS=linux
export GOARCH=amd64

docker build -t ${registryHost}registrator:${tag} -f Dockerfile .
docker push ${registryHost}registrator:${tag}

echo "docker Build complete!"