FROM golang:alpine as builder

WORKDIR /build

ENV CGO_ENABLED 0
ENV GOPROXY https://goproxy.cn,direct

COPY . .

RUN apk update --no-cache \
    && apk upgrade \
    && apk add --no-cache bash \
            bash-doc \
            bash-completion \
    && apk add --no-cache tzdata \
    && rm -rf /var/cache/apk/* \
    && go mod download \
    && bash ./scripts/build-all.sh

FROM alpine as prod

ENV TZ Asia/Shanghai

WORKDIR /work

RUN apk update --no-cache \
    && apk upgrade \
    && apk add yasm \
    && apk add ffmpeg \
    && rm -rf /var/cache/apk/*

COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai
COPY --from=builder /build/output .