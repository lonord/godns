FROM --platform=$BUILDPLATFORM golang:alpine AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

COPY . /app

ENV GOPROXY=https://goproxy.cn,direct
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

RUN cd /app \
    && go build -ldflags "-s -w" -o godns

FROM alpine

ENV TITLE=GODNS
ENV VERSION=0.1.2
ENV AUTHOR=kenshin
ENV DEBUG=false
ENV PORT=53
ENV SERVER_LIST_FILE=
ENV RESOLV_FILE=/etc/resolv.conf
ENV TIMEOUT=5
ENV INTERVAL=200
ENV SETEDNS0=false
ENV LOG_STDOUT=true
ENV LOG_FILE=
ENV LOG_LEVEL=INFO
ENV CACHE_BACKEND=memory
ENV CACHE_EXPIRE=600
ENV CACHE_MAXCOUNT=0
ENV CACHE_REFRESH=false
ENV CACHE_NO_NEGATIVE=false
ENV HOSTS_ENABLE=true
ENV HOSTS_HOST_FILE=/etc/hosts
ENV HOSTS_REDIS_ENABLE=false
ENV HOSTS_REDIS_KEY=godns:hosts
ENV HOSTS_TTL=600
ENV HOSTS_REFRESH_INTERVAL=5

COPY --from=builder /app/godns /app/godns
COPY --from=builder /app/docker-entrypoint.sh /docker-entrypoint.sh

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add ca-certificates bash \
    && rm -rf /var/cache/apk/* \
    && update-ca-certificates

RUN chmod +x /docker-entrypoint.sh

WORKDIR /app

EXPOSE 53/tcp
EXPOSE 53/udp

ENTRYPOINT [ "/docker-entrypoint.sh" ]