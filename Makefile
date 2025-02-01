version=

build:
	docker buildx build . --build-arg VERSION=${version} -t registry:local

local:
	docker run --rm -d -p 5050:5050 -e API_URL=http://localhost:5050 -e JWT_SECRET=qwerty --name registry registry:local
