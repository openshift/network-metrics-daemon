# Network Metrics Daemon

Network Metrics Daemon is a daemon component that collects and publishes network related metrics

## Rationale testing

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
- interface name (such as eth0)

This is fine until new interfaces are added to the pod, for example via [multus](https://github.com/intel/multus-cni), as it won't be clear what the interface names refers to.

The `interface` label refers to the interface name, but it's not clear what that interface is meant for. In case of many different interfaces, it would be impossible to understand what network the metrics we are monitoring refer to.

This is addressed by introducing the new `pod_network_name_info` described in the following section.

## Metrics with network name

This daemonset publishes a `pod_network_name_info` gauge metric, with a fixed value of 0:

```
pod_network_name_info{interface="net0",namespace="namespacename",network_name="nadnamespace/firstNAD",pod="podname"} 0
```

The network name label is produced using the annotation added by multus. It's the concatenation of the namespace the `network attachment definition` belongs to, plus the name of the `network attachment definition`.

The new metric alone does not provide much value, but combined with the container_network_* metrics mentioned above, it offers a better support for monitoring secondary networks.

Using a promql query like the following ones, it will be possible to get a new metric containing the value and the network name retrieved the `k8s.v1.cni.cncf.io/networks-status` annotation:

```
(container_network_receive_bytes_total) + on(namespace,pod,interface) group_left(network_name) ( pod_network_name_info )
(container_network_receive_errors_total) + on(namespace,pod,interface) group_left(network_name) ( pod_network_name_info )
(container_network_receive_packets_total) + on(namespace,pod,interface) group_left(network_name) ( pod_network_name_info )
(container_network_receive_packets_dropped_total) + on(namespace,pod,interface) group_left(network_name) ( pod_network_name_info )
(container_network_transmit_bytes_total) + on(namespace,pod,interface) group_left(network_name) ( pod_network_name_info )
(container_network_transmit_errors_total) + on(namespace,pod,interface) group_left(network_name) ( pod_network_name_info )
(container_network_transmit_packets_total) + on(namespace,pod,interface) group_left(network_name) ( pod_network_name_info )
(container_network_transmit_packets_dropped_total) + on(namespace,pod,interface) group_left(network_name) ( pod_network_name_info )
```

## Recording Rules

The new metrics can be produced also by applying a recording rule. Although this results in a more compact name to query, by adding the recording rule more resources are required as the query result is stored in prometheus. The recording rules for each metric can be found under [deployments/05_prometheus_rules.yaml](deployments/05_prometheus_rules.yaml).

## Architecture

This daemonset listens for the pods running on the same node it's running, finds the `k8s.v1.cni.cncf.io/networks-status` annotation and publishes a 0 value gauge with the pod name, the namespace and the network name.

## Deploy

Running `make deploy` will deploy the daemonset and set up the configuration to tie it to the Prometheus operator instance of an existing OpenShift 4+ cluster.

## Deploy on k8s

Running `make deploy-k8s` will deploy the daemonset and set up the configuration to tie it to the Prometheus operator instance of an existing kubernetes cluster where the prometheus operator was deployed. Please note that the k8s version exposes metrics over a plain http port.

## List of environment variables used in the tests

- METRIC_TEST_IMAGE - path to generic container image used in tests 