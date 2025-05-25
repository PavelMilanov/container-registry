version=

build:
	@docker buildx build . --builder=insecure-builder --build-arg VERSION=local-docker -t registry:local --cache-from type=local,src=./cache --cache-to type=local,dest=./cache --load

local: build
	@docker run --rm -d -p 5050:5050 -v ./src/conf.d:/registry/conf.d --name registry registry:local

release:
	@docker buildx build --platform linux/amd64 . --build-arg VERSION=${version} -t rosomilanov/container-registry:${version} -t rosomilanov/container-registry:latest

push: release
	@docker push rosomilanov/container-registry:${version}
	@docker push rosomilanov/container-registry:latest

buildx:
	@docker buildx build . --builder=insecure-builder -t 192.168.1.38:5050/dev/registry --cache-from type=local,src=./cache --cache-to type=local,dest=./cache --push
