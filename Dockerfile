FROM alpine:latest
RUN apk add libc6-compat

COPY armada-operator /

ENTRYPOINT ["/armada-operator"]
