# Docker Proxy

Call your unexposed container(s) without exposing it

## Installation

Run the container

```shell
docker run -it --name docker-proxy \
      -p 80:80 \
      anantadwi13/docker-proxy
```

or

```shell
docker run -it --name docker-proxy \
      -p 80:80 \
      --network network-name \
      anantadwi13/docker-proxy
```

## Usage

1. Make an http request
2. Set `X-Target-Host` header with the target host (ex: https://google.com or http://another-container)
3. Send that http request to `http://{hostname}/...`

Notes:

- Make sure this container has already connected to another network of the container you want to call