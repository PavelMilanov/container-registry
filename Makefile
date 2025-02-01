version=

build:
	docker buildx build . --build-arg VERSION=${version} -t registry:local

local:
	docker run --rm -d -p 8888:5050 -e API_URL=http://localhost:8888/api/ -e JWT_SECRET=qwerty --name registry registry:local
