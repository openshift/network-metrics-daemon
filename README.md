# network-metrics-daemon
network-metrics-daemon is a daemon component that collects and publishes network related metrics

## Rationale

The kubelet is already publishing network related metrics we can observe.
The set of metrics are:

- container_network_receive_bytes_total
- container_network_receive_errors_total
- container_network_receive_packets_total
- container_network_receive_packets_dropped_total
- container_network_transmit_bytes_total
- container_network_transmit_errors_total
- container_network_transmit_packets_total
- container_network_transmit_packets_dropped_total

The labels in these metrics contain (among others):

- pod name
- pod namespace
- interface name

This is fine until new interfaces are added to the pod, for example via [multus](https://github.com/intel/multus-cni), as it won't be clear what the interface names refers to.

## Metrics with network name

This daemonset publishes new metrics, containing the same information as the aforementioned ones, but with a new `network_name` label containing the name of the network. The network name is retrieved from the `k8s.v1.cni.cncf.io/networks-status` annotation.

The new set of metrics are named after:

- network:container_network_receive_bytes_total
- network:container_network_receive_bytes_total
- network:container_network_receive_errors_total
- network:container_network_receive_packets_total
- network:container_network_receive_packets_dropped_total
- network:container_network_transmit_bytes_total
- network:container_network_transmit_errors_total
- network:container_network_transmit_packets_total
- network:container_network_transmit_packets_dropped_total

and will make it possible to aggregate and set alarms on network classification basis.

## Architecture

This daemonset listens for the pods running on the same node it's running, finds the `k8s.v1.cni.cncf.io/networks-status` annotation and publishes a 0 value gauge with the pod name, the namespace and the network name.

A new recording rule is used to produce a new metric out of joining with the existing metrics produced by the kubelet.

This is achieved by joining the metrics produced by this daemon with the existing one, in the recording rules that can be found under [deployments/05_prometheus_rules.yaml](deployments/05_prometheus_rules.yaml).

## Deploy

Running `make deploy` will deploy the daemonset and set up the configuration to tie it to the Prometheus operator instance of an existing OpenShift 4+ cluster.
