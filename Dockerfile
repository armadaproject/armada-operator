FROM alpine:3.17

COPY armada-operator /

ENTRYPOINT ["/armada-operator"]
