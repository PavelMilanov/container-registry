version=

build:
	docker buildx build . --build-arg VERSION=${version} -t registry:local

local:
	docker run --rm -d -p 5050:5050 -e URL=http://localhost:5050 -e JWT_SECRET=qwerty --name registry registry:local

release:
	@docker buildx build --platform linux/amd64 . --build-arg VERSION=${version} -t rosomilanov/container-registry:${version} -t rosomilanov/container-registry:latest

push: release
	@docker push rosomilanov/container-registry:${version}
	@docker push rosomilanov/container-registry:latest