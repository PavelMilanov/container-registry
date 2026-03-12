version=

build:
	@docker buildx build . --builder=insecure-builder --build-arg VERSION=local-docker -t registry:local --cache-from type=local,src=./cache --cache-to type=local,dest=./cache --load

local: build
	@docker run -d -p 5050:5050 -v ./src/conf.d:/etc/conf.d -v ./src/var:/app/var/registry --name registry registry:local

release:
	@docker buildx build --platform linux/amd64 . --build-arg VERSION=${version} -t rosomilanov/container-registry:${version} -t rosomilanov/container-registry:latest

push: release
	@docker push rosomilanov/container-registry:${version}
	@docker push rosomilanov/container-registry:latest

buildx:
	@docker buildx build . --builder=insecure-builder --platform=linux/amd64 -t 192.168.11.100:5050/dev/registry --cache-from type=local,src=./cache --cache-to type=local,dest=./cache --push
