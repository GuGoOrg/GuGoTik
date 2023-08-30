FROM alpine:3.18.2

WORKDIR /clash
COPY ./static/clash /clash
COPY ./static/Country.mmdb /root/.config/clash


RUN apk update --no-cache \
    && apk upgrade \
    && rm -rf /var/cache/apk/*