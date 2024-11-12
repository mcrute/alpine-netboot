FROM alpine:latest
LABEL maintainer="Mike Crute <mike@pomonaconsulting.com>"

ENV TZ America/Los_Angeles

RUN set -euxo pipefail; \
    apk --no-cache add tzdata;

ADD bootstrap-server /usr/sbin/bootstrap-server

CMD ["/usr/sbin/bootstrap-server", "--debug"]
