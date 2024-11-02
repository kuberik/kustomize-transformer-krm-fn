FROM scratch
ARG binary
ENTRYPOINT ["/entrypoint"]
COPY $binary /entrypoint
