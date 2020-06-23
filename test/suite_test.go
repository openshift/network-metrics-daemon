package e2e_test

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testutils "github.com/openshift/network-metrics-daemon/test/utils"
	testclient "github.com/openshift/network-metrics-daemon/test/utils/client"
	"github.com/openshift/network-metrics-daemon/test/utils/namespaces"

	_ "github.com/openshift/network-metrics-daemon/test/network-metrics-daemon"
)

var junitPath *string

func init() {
	junitPath = flag.String("junit", "junit.xml", "the path for the junit format report")
}

// Test function
func Test(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	rr := []ginkgo.Reporter{}
	if junitPath != nil {
		rr = append(rr, reporters.NewJUnitReporter(*junitPath))
	}
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "Network Metrics Daemon e2e tests", rr)
}

var _ = ginkgo.BeforeSuite(func() {
	gomega.Expect(testclient.Client).NotTo(gomega.BeNil())

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testutils.NamespaceTesting,
		},
	}
	_, err := testclient.Client.Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
})

var _ = ginkgo.AfterSuite(func() {
	err := testclient.Client.Namespaces().Delete(context.Background(), testutils.NamespaceTesting, metav1.DeleteOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = namespaces.WaitForDeletion(testclient.Client, testutils.NamespaceTesting, 5*time.Minute)
})
