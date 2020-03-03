package podnetwork_test

import (
	"reflect"
	"testing"

	"github.com/openshift/network-metrics-daemon/pkg/podnetwork"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const simpleNetworkAnnotation = `[{
	"name": "kindnet",
	"interface": "eth0",
	"ips": [
		"10.244.0.10"
	],
	"mac": "4a:e9:0b:e2:63:67",
	"default": true,
	"dns": {}
}]`

const multipleNetworkAnnotation = `[{
	"name": "kindnet",
	"interface": "eth0",
	"ips": [
		"10.244.0.10"
	],
	"mac": "4a:e9:0b:e2:63:67",
	"default": true,
	"dns": {}
},{
	"name": "macvlan-conf",
	"interface": "net1",
	"ips": [
		"192.168.1.200"
	],
	"mac": "b2:07:4f:af:1c:a5",
	"dns": {}
}]`

const noAnnotation = ""

var podNetworkTests = []struct {
	testName string
	pod      *corev1.Pod
	res      []podnetwork.Network
}{
	{"defaultInterface",
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "PodName",
				Namespace: "PodNameSpace",
				Annotations: map[string]string{
					podnetwork.Status: simpleNetworkAnnotation,
				},
			},
		},
		[]podnetwork.Network{
			podnetwork.Network{
				Interface:   "eth0",
				NetworkName: "kindnet",
			},
		},
	},
	{"multipleinterfaces",
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "MultiplePodName",
				Namespace: "MultiplePodNameSpace",
				Annotations: map[string]string{
					podnetwork.Status: multipleNetworkAnnotation,
				},
			},
		},
		[]podnetwork.Network{
			podnetwork.Network{
				Interface:   "eth0",
				NetworkName: "kindnet",
			},
			podnetwork.Network{
				Interface:   "net1",
				NetworkName: "macvlan-conf",
			},
		},
	},
	{"nonetwork",
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "NoNetworkPodName",
				Namespace: "NoNetworkPodNamespace",
				Annotations: map[string]string{
					podnetwork.Status: noAnnotation,
				},
			},
		},
		[]podnetwork.Network{},
	},
}

func TestPodToNetwork(t *testing.T) {
	for _, tst := range podNetworkTests {
		networks, err := podnetwork.Get(tst.pod)
		if err != nil {
			t.Error(tst.testName, "Unexpected error", err)
			continue
		}
		if len(networks) != len(tst.res) {
			t.Error(tst.testName, "len(networks) != len(tst.res)", len(networks), len(tst.res))
			continue
		}
		if !reflect.DeepEqual(networks, tst.res) {
			t.Error(tst.testName, "Different result, expected", tst.res, "got", networks)
			continue
		}
	}
}
