FROM alpine

RUN apk add --no-cache bash curl docker-cli

COPY watchdog.sh /watchdog.sh

ENTRYPOINT ["/watchdog.sh"]
