package prometheus

import (
	"bytes"
	"context"

	"github.com/onsi/gomega"

	"github.com/openshift/network-metrics-daemon/test/utils/client"
	"github.com/openshift/network-metrics-daemon/test/utils/pods"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const openshiftMonitoringNamespace = "openshift-monitoring"

// Reply contains the Reply to a Prometheus query
type Reply struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				Container   string `json:"container"`
				Endpoint    string `json:"endpoint"`
				ID          string `json:"id"`
				Image       string `json:"image"`
				Instance    string `json:"instance"`
				Interface   string `json:"interface"`
				Job         string `json:"job"`
				MetricsPath string `json:"metrics_path"`
				Name        string `json:"name"`
				Namespace   string `json:"namespace"`
				NetworkName string `json:"network_name"`
				Node        string `json:"node"`
				Pod         string `json:"pod"`
				Service     string `json:"service"`
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// Query allows you to query prometheus
func Query(query string) bytes.Buffer {
	prometheusPods, err := client.Client.Pods(openshiftMonitoringNamespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: "app=prometheus",
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(len(prometheusPods.Items)).To(gomega.BeNumerically(">", 0))

	command := []string{"curl", query}
	stdout, err := pods.ExecCommand(client.Client, prometheusPods.Items[0], command)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	return stdout
}
