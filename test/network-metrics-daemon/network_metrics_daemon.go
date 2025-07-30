package networkmetrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/openshift/network-metrics-daemon/test/utils/client"
	"github.com/openshift/network-metrics-daemon/test/utils/consts"
	"github.com/openshift/network-metrics-daemon/test/utils/namespaces"
	"github.com/openshift/network-metrics-daemon/test/utils/pods"
	"github.com/openshift/network-metrics-daemon/test/utils/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	baseURL     = "http://localhost:9090"
	workerLabel = "node-role.kubernetes.io/worker="
)

var metrics = []string{
	"container_network_receive_bytes_total",
	"container_network_receive_errors_total",
	"container_network_receive_packets_total",
	"container_network_receive_packets_dropped_total",
	"container_network_transmit_bytes_total",
	"container_network_transmit_errors_total",
	"container_network_transmit_packets_total",
	"container_network_transmit_packets_dropped_total",
}

var _ = ginkgo.Describe("NetworkMetricsDaemon", func() {
	ginkgo.BeforeEach(func() {
		err := namespaces.Clean(consts.TestingNamespace, client.Client)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	ginkgo.Context("Network interface metrics", func() {
		ginkgo.BeforeEach(func() {
			ginkgo.By("creating the pod")

			metricsPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "metricpod",
					Namespace: consts.TestingNamespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "c1",
							Image:   consts.TestImage,
							Command: []string{"/bin/bash", "-c", "sleep inf"},
						},
					},
				},
			}

			metricsPod, err := client.Client.Pods(consts.TestingNamespace).Create(
				context.Background(),
				metricsPod,
				metav1.CreateOptions{},
			)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			ginkgo.By("waiting for the pod to be ready")
			gomega.Eventually(func() corev1.PodPhase {
				podObj, err := client.Client.Pods(consts.TestingNamespace).Get(context.Background(), metricsPod.Name, metav1.GetOptions{})
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				return podObj.Status.Phase
			}, 5*time.Minute, 5*time.Second).Should(gomega.Equal(corev1.PodRunning))
		})

		ginkgo.It("should be produced for the Pod's default interface", func() {
			query := fmt.Sprintf("pod_network_name_info{namespace=\"%s\",pod=\"metricpod\"}", consts.TestingNamespace)
			url := fmt.Sprintf("%s/api/v1/query?%s", baseURL, (url.Values{"query": []string{query}}).Encode())

			queryPrometheusEventually(url, 5*time.Minute, 5*time.Second, func(queryOutput prometheus.Reply) bool {
				result := queryOutput.Data.Results[0]
				return result.Value[1] == "0"
			})
		})

		ginkgo.It("should have the correct network_name and value on top of default pod network", func() {
			ginkgo.By("checking that there is traffic on all networks")

			recieveBytesMetric := fmt.Sprintf("container_network_receive_bytes_total{namespace=\"%s\",pod=\"metricpod\", interface=\"eth0\"}", consts.TestingNamespace)
			reciveBytesURL := fmt.Sprintf("%s/api/v1/query?%s", baseURL, (url.Values{"query": []string{recieveBytesMetric}}).Encode())

			queryPrometheusEventually(reciveBytesURL, 5*time.Minute, 5*time.Second, func(queryOutput prometheus.Reply) bool {
				for _, result := range queryOutput.Data.Results {
					if result.Value[1] == "0" {
						return false
					}
				}
				return true
			})

			transmitBytesMetric := fmt.Sprintf("container_network_transmit_bytes_total{namespace=\"%s\",pod=\"metricpod\", interface=\"eth0\"}", consts.TestingNamespace)
			transmitBytesURL := fmt.Sprintf("%s/api/v1/query?%s", baseURL, (url.Values{"query": []string{transmitBytesMetric}}).Encode())

			queryPrometheusEventually(transmitBytesURL, 5*time.Minute, 5*time.Second, func(queryOutput prometheus.Reply) bool {
				for _, result := range queryOutput.Data.Results {
					if result.Value[1] == "0" {
						return false
					}
				}
				return true
			})

			ginkgo.By("checking that difference in traffic is 0")

			differenceQuery := fmt.Sprintf(
				"((%s) + on(namespace,pod,interface) group_left(network_name) (pod_network_name_info{namespace=\"%s\",pod=\"metricpod\"})) - ignoring(network_name) %s{namespace=\"%s\",pod=\"metricpod\", interface=\"eth0\"}",
				"%s",
				consts.TestingNamespace,
				"%s",
				consts.TestingNamespace,
			)

			for _, metric := range metrics {
				currentQuery := fmt.Sprintf(differenceQuery, metric, metric)
				url := fmt.Sprintf("%s/api/v1/query?%s", baseURL, (url.Values{"query": []string{currentQuery}}).Encode())

				queryPrometheusEventually(url, 5*time.Minute, 5*time.Second, func(queryOutput prometheus.Reply) bool {
					for _, result := range queryOutput.Data.Results {
						if result.Value[1] != "0" {
							return false
						}
					}
					return true
				})
			}
		})
	})

	ginkgo.Context("Network Name metric", func() {
		// workerName is used to create all the pods explicitly on the same worker node
		var workerName string = ""

		ginkgo.BeforeEach(func() {
			if workerName == "" {
				workerList, err := client.Client.Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: workerLabel})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(len(workerList.Items)).To(gomega.BeNumerically(">", 0))
				workerName = workerList.Items[0].Name
			}

			networkAttachmentDefinition0 := &nettypes.NetworkAttachmentDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nad0",
					Namespace: consts.TestingNamespace,
				},
				Spec: nettypes.NetworkAttachmentDefinitionSpec{
					Config: `{
								"cniVersion": "0.3.0",
								"type": "macvlan",
								"mode": "bridge",
								"ipam": {
									"type": "host-local",
									"ranges": [
										[ {
											"subnet": "192.168.200.0/24",
											"rangeStart": "192.168.200.10",
											"rangeEnd": "192.168.200.200"
										} ]
									]
								}
							}`,
				},
			}
			networkAttachmentDefinition1 := &nettypes.NetworkAttachmentDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nad1",
					Namespace: consts.TestingNamespace,
				},
				Spec: nettypes.NetworkAttachmentDefinitionSpec{
					Config: `{
								"cniVersion": "0.3.0",
								"type": "macvlan",
								"mode": "bridge",
								"ipam": {
									"type": "host-local",
									"ranges": [
										[ {
											"subnet": "192.168.202.0/24",
											"rangeStart": "192.168.202.10",
											"rangeEnd": "192.168.202.200"
										} ]
									]
								}
							}`,
				},
			}

			_, err := client.Client.NetworkAttachmentDefinitions(consts.TestingNamespace).Create(
				context.Background(),
				networkAttachmentDefinition0,
				metav1.CreateOptions{},
			)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			_, err = client.Client.NetworkAttachmentDefinitions(consts.TestingNamespace).Create(
				context.Background(),
				networkAttachmentDefinition1,
				metav1.CreateOptions{},
			)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"k8s.v1.cni.cncf.io/networks": fmt.Sprintf("%s/nad0, %s/nad1", consts.TestingNamespace, consts.TestingNamespace),
					},
					Name:      "metricpod",
					Namespace: consts.TestingNamespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "c1",
							Image:   consts.TestImage,
							Command: []string{"/bin/bash", "-c", "sleep inf"},
						},
					},
					NodeName: workerName,
				},
			}

			pod, err = client.Client.Pods(consts.TestingNamespace).Create(
				context.Background(),
				pod,
				metav1.CreateOptions{},
			)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Eventually(func() corev1.PodPhase {
				podObj, err := client.Client.Pods(consts.TestingNamespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				return podObj.Status.Phase
			}, 5*time.Minute, 5*time.Second).Should(gomega.Equal(corev1.PodRunning))
		})

		ginkgo.It("should have the correct network_name", func() {
			query := fmt.Sprintf("(pod_network_name_info{namespace=\"%s\",pod=\"metricpod\"})", consts.TestingNamespace)
			url := fmt.Sprintf("%s/api/v1/query?%s", baseURL, (url.Values{"query": []string{query}}).Encode())

			queryPrometheusEventually(url, 5*time.Minute, 5*time.Second, func(queryOutput prometheus.Reply) bool {
				results := queryOutput.Data.Results

				// Check that we have exactly 3 results (default network + 2 additional networks)
				if len(results) != 3 {
					return false
				}

				// Track which networks we've found
				foundNetworks := make(map[string]bool)

				// Validate each result
				for _, result := range results {
					if result.Metric.NetworkName != "" {
						foundNetworks[result.Metric.NetworkName] = true
					}
				}

				// Check that we have both expected networks
				expectedNad0 := fmt.Sprintf("%s/nad0", consts.TestingNamespace)
				expectedNad1 := fmt.Sprintf("%s/nad1", consts.TestingNamespace)

				return foundNetworks[expectedNad0] && foundNetworks[expectedNad1]
			})
		})

		ginkgo.It("should return the same values for the new query as the standard query", func() {
			testPod, err := client.Client.Pods(consts.TestingNamespace).Get(context.Background(), "metricpod", metav1.GetOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			podIP := testPod.Status.PodIP
			gomega.Expect(podIP).ToNot(gomega.BeEmpty())

			pingPod(podIP, workerName, fmt.Sprintf("%s/nad0", consts.TestingNamespace))
			pingPod(podIP, workerName, fmt.Sprintf("%s/nad1", consts.TestingNamespace))

			ginkgo.By("checking that there is traffic on all networks")

			recieveBytesMetric := fmt.Sprintf("(container_network_receive_bytes_total{namespace=\"%s\",pod=\"metricpod\", interface=\"net1\"})", consts.TestingNamespace)
			reciveBytesURL := fmt.Sprintf("%s/api/v1/query?%s", baseURL, (url.Values{"query": []string{recieveBytesMetric}}).Encode())

			queryPrometheusEventually(reciveBytesURL, 5*time.Minute, 5*time.Second, func(queryOutput prometheus.Reply) bool {
				for _, result := range queryOutput.Data.Results {
					if result.Value[1] == "0" {
						return false
					}
				}
				return true
			})

			transmitBytesMetric := fmt.Sprintf("(container_network_transmit_bytes_total{namespace=\"%s\",pod=\"metricpod\", interface=\"net1\"})", consts.TestingNamespace)
			transmitBytesURL := fmt.Sprintf("%s/api/v1/query?%s", baseURL, (url.Values{"query": []string{transmitBytesMetric}}).Encode())

			queryPrometheusEventually(transmitBytesURL, 5*time.Minute, 5*time.Second, func(queryOutput prometheus.Reply) bool {
				for _, result := range queryOutput.Data.Results {
					if result.Value[1] == "0" {
						return false
					}
				}
				return true
			})

			ginkgo.By("checking that difference in traffic is 0")

			// for testing in web console :
			// ---------------------------
			// differenceQuery = ((container_network_receive_bytes_total) + on(namespace,pod,interface) group_left(network_name) (pod_network_name_info{namespace="metrictest",pod="metricpod",network_name="nad0", interface="net1"})) - ignoring(network_name) (container_network_receive_bytes_total{namespace="metrictest",pod="metricpod", interface="net1"})

			differenceQuery := fmt.Sprintf(
				"((%s) + on(namespace,pod,interface) group_left(network_name) (pod_network_name_info{namespace=\"%s\",pod=\"metricpod\",network_name=\"%s/nad0\", interface=\"net1\"})) - ignoring(network_name) (%s{namespace=\"%s\",pod=\"metricpod\", interface=\"net1\"})",
				"%s",
				consts.TestingNamespace,
				consts.TestingNamespace,
				"%s",
				consts.TestingNamespace,
			)

			for _, metric := range metrics {
				currentQuery := fmt.Sprintf(differenceQuery, metric, metric)
				url := fmt.Sprintf("%s/api/v1/query?%s", baseURL, (url.Values{"query": []string{currentQuery}}).Encode())

				queryPrometheusEventually(url, 5*time.Minute, 5*time.Second, func(queryOutput prometheus.Reply) bool {
					for _, result := range queryOutput.Data.Results {
						if result.Value[1] != "0" {
							return false
						}
					}
					return true
				})
			}
		})
	})
})

func queryPrometheusEventually(query string, total time.Duration, interval time.Duration, validate func(queryOutput prometheus.Reply) bool) {
	var queryOutput prometheus.Reply
	gomega.Eventually(func() bool {
		jsonReply, err := prometheus.Query(query)
		if err != nil {
			return false
		}
		err = json.Unmarshal([]byte(jsonReply.String()), &queryOutput)
		if err != nil {
			return false
		}
		if queryOutput.Status != "success" {
			return false
		}
		if len(queryOutput.Data.Results) <= 0 {
			return false
		}
		return validate(queryOutput)
	}, total, interval).Should(gomega.BeTrue())
}

func pingPod(ip string, nodeName string, networkAttachmentDefinition string) {
	podDefinition := pods.RedifineWithSpecificNode(
		pods.RedefineWithRestartPolicy(
			pods.RedefineWithCommand(
				pods.DefineWithNetworks([]string{networkAttachmentDefinition}),
				[]string{"/bin/bash", "-c", "ping -c 3 " + ip}, []string{},
			),
			corev1.RestartPolicyNever,
		),
		nodeName,
	)
	createdPod, err := client.Client.Pods(consts.TestingNamespace).Create(context.Background(), podDefinition, metav1.CreateOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	gomega.Eventually(func() corev1.PodPhase {
		runningPod, err := client.Client.Pods(consts.TestingNamespace).Get(context.Background(), createdPod.Name, metav1.GetOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		return runningPod.Status.Phase
	}, 5*time.Minute, 5*time.Second).Should(gomega.Equal(corev1.PodSucceeded))
}
