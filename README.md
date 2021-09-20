# Docker Proxy

Call your unexposed container(s) without exposing it

## Features

- TCP Proxy (Layer 4)
- HTTP Proxy (Layer 7)
- Dynamically set target host via HTTP request header

## Installation

### Option 1 (direct)

1. Install from source
   ```shell
   go install github.com/anantadwi13/docker-proxy
   ```
2. Run docker-proxy server locally
   ```shell
   docker-proxy
   # or
   docker-proxy --mode http --target-host https://github.com/
   # or
   docker-proxy --mode tcp --local-address :9999 --remote-address localhost:3306
   ```

### Option 2 (as container)

1. Download image from Docker Hub
   ```shell
   docker pull anantadwi13/docker-proxy
   ```
2. Run docker-proxy server via docker
   ```shell
   docker run -it --name docker-proxy \
          -p 80:80 \
          anantadwi13/docker-proxy
   # or
   docker run -it --name docker-proxy \
          -p 80:80 \
          anantadwi13/docker-proxy --target-host https://github.com/
   # or
   docker run -it --name docker-proxy \
          -p 9999:9999 \
          anantadwi13/docker-proxy --mode tcp --local-address :9999 --remote-address mysql-server:3306
   # or
   docker run -it --name docker-proxy \
          -p 80:80 \
          --network network-name \
          anantadwi13/docker-proxy
   ```

## Usage

1. Make sure docker-proxy server is already running
2. Make an HTTP request or TCP connection
3. (Optional for HTTP mode) Set `X-Target-Host` header with the target host (ex: https://google.com
   or http://another-container)
4. Send this HTTP request to `http://{docker_proxy_local_address}/...` or TCP connection to `docker_proxy_local_address`

Notes:

- Make sure docker-proxy is running as a container and connected to particular network if you want to proxy other
  containers

## Available Arguments

```shell
$ docker-proxy --help
Usage of docker-proxy:
  -l, --local-address string    local address (default ":80")
  -m, --mode string             proxy mode, http or tcp (default "http")
  -r, --remote-address string   remote address, required for tcp mode
  -t, --target-host string      target host, optional for http mode
```