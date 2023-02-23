# build and publish docker image: spartahasura/supervisord:v0.7.3 with multi-arch
docker buildx build . -f Dockerfile \
  --platform linux/arm64/v8,linux/amd64 \
  --tag spartahasura/supervisord:v0.7.3 \
  --push
