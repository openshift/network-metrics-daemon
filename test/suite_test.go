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

	"github.com/openshift/network-metrics-daemon/test/utils/client"
	"github.com/openshift/network-metrics-daemon/test/utils/consts"
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
	gomega.Expect(client.Client).NotTo(gomega.BeNil())

	nameSpace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: consts.TestingNamespace,
			Labels: map[string]string{
				"pod-security.kubernetes.io/audit":               "privileged",
				"pod-security.kubernetes.io/enforce":             "privileged",
				"pod-security.kubernetes.io/warn":                "privileged",
				"security.openshift.io/scc.podSecurityLabelSync": "false",
			},
		},
	}
	_, err := client.Client.Namespaces().Create(context.Background(), nameSpace, metav1.CreateOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
})

var _ = ginkgo.AfterSuite(func() {
	err := client.Client.Namespaces().Delete(context.Background(), consts.TestingNamespace, metav1.DeleteOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = namespaces.WaitForDeletion(client.Client, consts.TestingNamespace, 10*time.Minute)
})
