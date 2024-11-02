FROM alpine:3.15
ARG binary
ENTRYPOINT ["/entrypoint"]
COPY $binary /entrypoint
