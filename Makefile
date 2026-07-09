version=
image=rosomilanov/container-registry
platform=linux/amd64

.PHONY: build local release push buildx

build:
	@docker buildx build . --builder=insecure-builder --build-arg VERSION=local-docker -t registry:local --cache-from type=local,src=./cache --cache-to type=local,dest=./cache --load

local: build
	@docker run -d -p 5050:5050 -v ./src/conf.d:/etc/conf.d -v ./src/var:/app/var/registry --name registry registry:local

push:
	@docker buildx build --platform ${platform} . --build-arg VERSION=${version} -t ${image}:${version} -t ${image}:latest --push

buildx:
	@docker buildx build . --builder=insecure-builder --platform=linux/amd64 -t 192.168.11.100:5050/dev/registry --cache-from type=local,src=./cache --cache-to type=local,dest=./cache --push
