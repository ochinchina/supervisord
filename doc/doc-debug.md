# Doc for Debugging

## Debugging the Docker Image

```shell
docker run --rm -it \
  --name app \
  -p 19001:9001 \
  -p 10080:80 \
  -v $(pwd):/data \
  --entrypoint=/bin/bash \
  qpod/supervisord:ubuntu


mkdir -pv /var/log/supervisord
/opt/supervisord/supervisord -c ./supervisord.conf
```
