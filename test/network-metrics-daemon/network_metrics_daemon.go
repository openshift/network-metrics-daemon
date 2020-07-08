package networkmetrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

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
							Name:    "sleep-inf",
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
			}, 5*time.Minute, time.Second).Should(gomega.Equal(corev1.PodRunning))
		})

		ginkgo.It("should be produced for the Pod's default interface", func() {
			query := "pod_network_name_info{namespace=\"metrictest\",pod=\"metricpod\"}"
			url := fmt.Sprintf("%s/api/v1/query?%s", baseURL, (url.Values{"query": []string{query}}).Encode())

			jsonReply := prometheus.Query(url)

			var queryOutput prometheus.Reply
			err := json.Unmarshal([]byte(jsonReply.String()), &queryOutput)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(queryOutput.Status).To(gomega.Equal("success"))
			gomega.Expect(len(queryOutput.Data.Result)).To(gomega.BeNumerically(">", 0))

			result := queryOutput.Data.Result[0]
			gomega.Expect(result.Value[1]).To(gomega.Equal("0"))
		})
	})
})
