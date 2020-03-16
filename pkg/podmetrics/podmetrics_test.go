package podmetrics_test

import (
	"strings"
	"testing"

	"github.com/openshift/network-metrics-daemon/pkg/podmetrics"
	"github.com/openshift/network-metrics-daemon/pkg/podnetwork"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

var podMetricsTests = []struct {
	testName        string
	setMetrics      func()
	expectedMetrics string
}{
	{
		"twonetworks same network name",
		func() {
			networks := []podnetwork.Network{
				{"eth0", "firstNAD"},
				{"eth1", "firstNAD"},
			}
			podmetrics.UpdateForPod("podname", "namespacename", networks)
		},
		`
			pod_network_name_info{interface="eth0",network_name="firstNAD",source_namespace="namespacename",source_pod="podname"} 0
			pod_network_name_info{interface="eth1",network_name="firstNAD",source_namespace="namespacename",source_pod="podname"} 0
			`,
	},
	{
		"twonetworks different networkname",
		func() {
			networks := []podnetwork.Network{
				{"eth0", "firstNAD"},
				{"eth1", "secondNAD"},
			}
			podmetrics.UpdateForPod("podname", "namespacename", networks)
		},
		`
			pod_network_name_info{interface="eth0",network_name="firstNAD",source_namespace="namespacename",source_pod="podname"} 0
			pod_network_name_info{interface="eth1",network_name="secondNAD",source_namespace="namespacename",source_pod="podname"} 0
			`,
	},
	{
		"add and delete",
		func() {
			networks := []podnetwork.Network{
				{"eth0", "firstNAD"},
				{"eth1", "secondNAD"},
			}
			podmetrics.UpdateForPod("podname", "namespacename", networks)
			podmetrics.DeleteAllForPod("podname", "namespacename")
		},
		`
		`,
	},
	{
		"two pods and delete one",
		func() {
			networks := []podnetwork.Network{
				{"eth0", "firstNAD"},
				{"eth1", "secondNAD"},
			}
			networks2 := []podnetwork.Network{
				{"eth0", "firstNAD"},
			}
			podmetrics.UpdateForPod("podname1", "namespacename", networks)
			podmetrics.UpdateForPod("podname2", "namespacename", networks2)
			podmetrics.DeleteAllForPod("podname1", "namespacename")

		},
		`
			pod_network_name_info{interface="eth0",network_name="firstNAD",source_namespace="namespacename",source_pod="podname2"} 0

		`,
	},
}

func TestPodMetrics(t *testing.T) {

	const metadata = `
	# HELP pod_network_name_info Metric to identify network names of networks added to pods.
	# TYPE pod_network_name_info gauge
	`

	for _, tst := range podMetricsTests {
		tst.setMetrics()
		err := testutil.CollectAndCompare(podmetrics.NetAttachDefPerPod, strings.NewReader(metadata+tst.expectedMetrics))
		if err != nil {
			t.Error("Failed to collect metrics", tst.testName, err)
		}
		podmetrics.NetAttachDefPerPod.Reset()
	}

}
