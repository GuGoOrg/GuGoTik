FROM golang:alpine as builder

WORKDIR /build

ENV CGO_ENABLED 0
ENV GOPROXY https://goproxy.cn/,direct

RUN apk update --no-cache \
    && apk upgrade \
    && apk add --no-cache bash \
            bash-doc \
            bash-completion \
    && apk add --no-cache tzdata \
    && rm -rf /var/cache/apk/*

COPY . .

RUN go mod download \
    && bash ./scripts/build-all.sh

FROM docker.io/epicmo/gugotik-basic:1.3 as prod

ENV TZ Asia/Shanghai

WORKDIR /data/apps/gugotik-service-bundle

RUN apk update --no-cache \
    && apk upgrade

COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
COPY --from=builder /build/output .