package networkmetrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/openshift/network-metrics-daemon/test/utils/client"
	"github.com/openshift/network-metrics-daemon/test/utils/consts"
	"github.com/openshift/network-metrics-daemon/test/utils/namespaces"
	"github.com/openshift/network-metrics-daemon/test/utils/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	baseURL string = "http://localhost:9090"
)

var _ = ginkgo.Describe("NetworkMetricsDaemon", func() {
	ginkgo.BeforeEach(func() {
		err := namespaces.Clean(consts.NamespaceTesting, client.Client)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	ginkgo.It("should check the client connected", func() {
		ginkgo.By("checking that the client is connected")
		gomega.Expect(client.Client).ToNot(gomega.BeNil())
	})

	ginkgo.Context("Network interface metrics", func() {
		ginkgo.BeforeEach(func() {
			ginkgo.By("creating the pod")

			metricsPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "metricpod",
					Namespace: consts.NamespaceTesting,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "c1",
							Image:   "centos",
							Command: []string{"/bin/bash", "-c", "sleep inf"},
						},
					},
				},
			}

			metricsPod, err := client.Client.Pods(consts.NamespaceTesting).Create(
				context.Background(),
				metricsPod,
				metav1.CreateOptions{},
			)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			ginkgo.By("waiting for the pod to be ready")
			gomega.Eventually(func() corev1.PodPhase {
				podObj, err := client.Client.Pods(consts.NamespaceTesting).Get(context.Background(), metricsPod.Name, metav1.GetOptions{})
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				return podObj.Status.Phase
			}, 30*time.Minute, 10*time.Second).Should(gomega.Equal(corev1.PodRunning))
		})

		ginkgo.It("should be produced for the Pod's default interface", func() {
			query := "pod_network_name_info{namespace=\"metrictest\",pod=\"metricpod\"}"
			url := fmt.Sprintf("%s/api/v1/query?%s", baseURL, (url.Values{"query": []string{query}}).Encode())

			queryOutput := queryPrometheusEventually(url, 30*time.Minute, 10*time.Second)

			result := queryOutput.Data.Results[0]
			gomega.Expect(result.Value[1]).To(gomega.Equal("0"))
		})

		ginkgo.It("should have the correct network_name and value on top of default pod network", func() {
			differenceQuery := "((%s) + on(namespace,pod,interface) group_left(network_name) (pod_network_name_info{namespace=\"metrictest\",pod=\"metricpod\"})) - ignoring(network_name) %s{namespace=\"metrictest\",pod=\"metricpod\", interface=\"eth0\"}"

			queries := []string{
				"container_network_receive_bytes_total",
				"container_network_receive_errors_total",
				"container_network_receive_packets_total",
				"container_network_receive_packets_dropped_total",
				"container_network_transmit_bytes_total",
				"container_network_transmit_errors_total",
				"container_network_transmit_packets_total",
				"container_network_transmit_packets_dropped_total",
			}

			for _, query := range queries {
				currentQuery := fmt.Sprintf(differenceQuery, query, query)
				url := fmt.Sprintf("%s/api/v1/query?%s", baseURL, (url.Values{"query": []string{currentQuery}}).Encode())

				queryOutput := queryPrometheusEventually(url, 30*time.Minute, 10*time.Second)

				for _, result := range queryOutput.Data.Results {
					gomega.Expect(result.Value[1]).To(gomega.Equal("0"))
				}
			}
		})
	})

	ginkgo.It("Network_attachment_definitions configuration", func() {
		networkAttachmentDefinition0 := &nettypes.NetworkAttachmentDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nad",
				Namespace: consts.NamespaceTesting,
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
				Name:      "nad2",
				Namespace: consts.NamespaceTesting,
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

		_, err := client.Client.NetworkAttachmentDefinitions("metrictest").Create(
			context.Background(),
			networkAttachmentDefinition0,
			metav1.CreateOptions{},
		)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		_, err = client.Client.NetworkAttachmentDefinitions("metrictest").Create(
			context.Background(),
			networkAttachmentDefinition1,
			metav1.CreateOptions{},
		)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	ginkgo.Context("Network Name metric", func() {
		ginkgo.BeforeEach(func() {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"k8s.v1.cni.cncf.io/networks": "metrictest/nad, metrictest/nad2",
					},
					Name:      "metricpod",
					Namespace: consts.NamespaceTesting,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "c1",
							Image:   "centos",
							Command: []string{"/bin/bash", "-c", "sleep inf"},
						},
					},
				},
			}

			pod, err := client.Client.Pods("metrictest").Create(
				context.Background(),
				pod,
				metav1.CreateOptions{},
			)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Eventually(func() corev1.PodPhase {
				podObj, err := client.Client.Pods(consts.NamespaceTesting).Get(context.Background(), pod.Name, metav1.GetOptions{})
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				return podObj.Status.Phase
			}, 30*time.Minute, 10*time.Second).Should(gomega.Equal(corev1.PodRunning))
		})

		ginkgo.It("should have the correct network_name", func() {
			query := "(pod_network_name_info{namespace=\"metrictest\",pod=\"metricpod\"})"
			url := fmt.Sprintf("%s/api/v1/query?%s", baseURL, (url.Values{"query": []string{query}}).Encode())

			queryOutput := queryPrometheusEventually(url, 30*time.Minute, 10*time.Second)

			results := queryOutput.Data.Results
			gomega.Expect(len(results)).To(gomega.BeNumerically(">=", 2))

			gomega.Expect(results).To(gomega.ContainElement(gstruct.MatchFields(
				gstruct.IgnoreExtras,
				gstruct.Fields{
					"Metric": gstruct.MatchFields(
						gstruct.IgnoreExtras,
						gstruct.Fields{
							"NetworkName": gomega.Equal("nad"),
						},
					),
				},
			)))

			gomega.Expect(results).To(gomega.ContainElement(gstruct.MatchFields(
				gstruct.IgnoreExtras,
				gstruct.Fields{
					"Metric": gstruct.MatchFields(
						gstruct.IgnoreExtras,
						gstruct.Fields{
							"NetworkName": gomega.Equal("nad2"),
						},
					),
				},
			)))
		})
	})
})

func queryPrometheusEventually(query string, total time.Duration, interval time.Duration) (queryOutput prometheus.Reply) {
	gomega.Eventually(func() error {
		jsonReply, err := prometheus.Query(query)
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(jsonReply.String()), &queryOutput)
		if err != nil {
			return err
		}
		if queryOutput.Status != "success" {
			return errors.New("query failed")
		}
		if len(queryOutput.Data.Results) <= 0 {
			return errors.New("no results")
		}
		return nil
	}, total, interval).ShouldNot(gomega.HaveOccurred())

	return queryOutput
}
