version=

build:
	docker buildx build . --build-arg VERSION=dev -t registry:local

local: build
	docker run --rm -d -p 5050:5050 -v ./src/conf.d:/registry/conf.d --name registry registry:local

release:
	@docker buildx build --platform linux/amd64 . --build-arg VERSION=${version} -t rosomilanov/container-registry:${version} -t rosomilanov/container-registry:latest

push: release
	@docker push rosomilanov/container-registry:${version}
	@docker push rosomilanov/container-registry:latest
