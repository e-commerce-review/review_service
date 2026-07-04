FROM golang:1.26 AS builder
ARG VERSION=dev
ARG SERVICE_NAME=review_service

COPY . /src
WORKDIR /src

RUN mkdir -p bin/ && GOPROXY=https://goproxy.cn go build \
	-ldflags "-X main.Version=${VERSION} -X main.Name=${SERVICE_NAME}" \
	-o ./bin/ ./...

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/bin /app

WORKDIR /app

EXPOSE 8000
EXPOSE 9000
VOLUME /data/conf

CMD ["./review_service", "-conf", "/data/conf"]
