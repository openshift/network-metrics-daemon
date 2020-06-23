package networkmetrics

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	testclient "github.com/openshift/network-metrics-daemon/test/utils/client"
)

var _ = ginkgo.Describe("NetworkMetricsDaemon", func() {
	ginkgo.It("should check the client connected", func() {
		gomega.Expect(testclient.Client).ToNot(gomega.BeNil())
	})
})
