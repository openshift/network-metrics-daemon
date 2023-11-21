package consts

import "os"

// TestImage contains test image (default is quay.io/centos/centos)
var TestImage string

func init() {
	TestImage = os.Getenv("METRIC_TEST_IMAGE")
	if TestImage == "" {
		TestImage = "quay.io/centos/centos"
	}
}

const (
	// TestingNamespace contains the name of the testing namespace
	TestingNamespace = "metrictest"
)
