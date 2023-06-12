FROM golang:1.20 AS builder
WORKDIR /go/src/github.com/openshift/network-metrics-daemon
COPY . .
RUN make build-bin

FROM centos:7
LABEL io.k8s.display-name="network-metrics-daemon" \
    io.k8s.description="This is a daemon exposing network related metrics"
COPY --from=builder /go/src/github.com/openshift/network-metrics-daemon/bin/network-metrics-daemon /usr/bin/network-metrics
CMD ["/usr/bin/network-metrics"]

