package consts

import "os"

// TestImage contains test image (default is k8s.gcr.io/e2e-test-images/agnhost:2.47)
var TestImage string

func init() {
	TestImage = os.Getenv("METRIC_TEST_IMAGE")
	if TestImage == "" {
		TestImage = "k8s.gcr.io/e2e-test-images/agnhost:2.47"
	}
}

const (
	// TestingNamespace contains the name of the testing namespace
	TestingNamespace = "metrictest"
)
