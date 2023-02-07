FROM scratch

COPY armada-operator /

ENTRYPOINT ["/armada-operator"]
