# syntax=docker/dockerfile:1.4

FROM alpine:3.16

LABEL org.opencontainers.image.authors="Exograd"
LABEL org.opencontainers.image.url="https://www.eventline.net"
LABEL org.opencontainers.image.documentation="https://www.eventline.net/docs"
LABEL org.opencontainers.image.source="https://github.com/exograd/eventline"
LABEL org.opencontainers.image.vendor="Exograd"
LABEL org.opencontainers.image.licenses="ISC"
LABEL org.opencontainers.image.title="Eventline"
LABEL org.opencontainers.image.description="Job scheduling platform."

ENV LANG en_US.utf8

RUN <<EOF
    set -eu

    apk --no-cache add curl postgresql14-client

    addgroup -S eventline
    adduser -G eventline -g Eventline -H -D eventline
EOF

USER eventline:eventline

COPY --chown=eventline:eventline bin/* /usr/bin/

COPY --chown=eventline:eventline data/assets/ /usr/share/eventline/assets/
COPY --chown=eventline:eventline data/pg/ /usr/share/eventline/pg/
COPY --chown=eventline:eventline data/templates/ /usr/share/eventline/templates/

COPY --chown=eventline:eventline --chmod=0600 \
    docker/eventline.yaml /etc/eventline/eventline.yaml

COPY docker/entrypoint.sh /usr/bin/entrypoint

ENTRYPOINT ["entrypoint"]

EXPOSE 8085/TCP 8087/TCP

HEALTHCHECK --start-period=5s --interval=1m --timeout=5s --retries=3 \
    CMD curl -I -f http://localhost:8085/status

CMD ["eventline", "-c", "/etc/eventline/eventline.yaml"]
