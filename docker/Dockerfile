FROM alpine

RUN apk update --no-cache \
    && apk upgrade \
    && apk add yasm \
    && apk add ffmpeg \
    && rm -rf /var/cache/apk/*