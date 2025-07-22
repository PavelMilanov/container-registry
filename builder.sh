# Create buildkitd.toml
echo '[registry."192.168.1.38:5050"]
  http = true
  insecure = true' > buildkitd.toml

docker buildx create \
  --name insecure-builder \
  --driver docker-container \
  --driver-opt network=host \
  --config buildkitd.toml \
