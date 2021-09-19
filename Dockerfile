FROM golang:1.17 AS builder

WORKDIR /go/src/proxy
COPY go.* ./
RUN go mod download
COPY cmd cmd
COPY *.go .
RUN go mod tidy
RUN GOOS=linux CGO_ENABLED=0 go build -o service ./cmd/docker-proxy

FROM alpine:3.14
WORKDIR /root
COPY --from=builder /go/src/proxy/service .

EXPOSE 80

CMD ./service