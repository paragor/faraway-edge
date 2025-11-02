FROM alpine:3.20.2

WORKDIR /app

COPY faraway-edge /usr/bin/
ENTRYPOINT ["/usr/bin/faraway-edge"]
