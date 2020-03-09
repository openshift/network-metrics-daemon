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
			network_attachment_definition_per_pod{interface="eth0",metricsnamespace="namespacename",metricspod="podname",networkname="firstNAD"} 0
			network_attachment_definition_per_pod{interface="eth1",metricsnamespace="namespacename",metricspod="podname",networkname="firstNAD"} 0
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
			network_attachment_definition_per_pod{interface="eth0",metricsnamespace="namespacename",metricspod="podname",networkname="firstNAD"} 0
			network_attachment_definition_per_pod{interface="eth1",metricsnamespace="namespacename",metricspod="podname",networkname="secondNAD"} 0
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
			network_attachment_definition_per_pod{interface="eth0",metricsnamespace="namespacename",metricspod="podname2",networkname="firstNAD"} 0

		`,
	},
}

func TestPodMetrics(t *testing.T) {

	const metadata = `
	# HELP network_attachment_definition_per_pod Metric to identify clusters with network attachment definition enabled instances.
	# TYPE network_attachment_definition_per_pod gauge
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
