#!/bin/bash

ip=172.26.148.189
tag=$(grep "^version" docker_run.config | awk -F'=' '{print $2}')
id=myRegistrator
dockerRun=qa.haidao:5000/registrator:${tag}

docker stop $id && docker rm $id && echo "stop and remove $id"
docker run -d  --name=$id  --net=host  --volume=/var/run/docker.sock:/tmp/docker.sock  $dockerRun  \
-ip="$ip" \
-useIpFromLabel="exposeIP" \
-ttl=60 \
-ttl-refresh=30  \
-event-channel-len=1024 \
etcd://qa.haidao:2379/services