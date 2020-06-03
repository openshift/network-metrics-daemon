package controller

import (
	"strings"
	"testing"
	"time"

	promtestutil "github.com/prometheus/client_golang/prometheus/testutil"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/openshift/network-metrics-daemon/pkg/podmetrics"
	"github.com/openshift/network-metrics-daemon/pkg/podnetwork"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

const metadata = `
	# HELP pod_network_name_info Metric to identify network names of networks added to pods.
	# TYPE pod_network_name_info gauge
	`

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
	t               *testing.T
	kubeclient      *k8sfake.Clientset
	podsLister      []*v1.Pod
	kubeobjects     []runtime.Object
	expectedMetrics string
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.kubeobjects = []runtime.Object{}
	return f
}

func newPod(name, namespace string, networkAnnotation string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				podnetwork.Status: networkAnnotation,
			},
		},
		Spec: v1.PodSpec{
			NodeName: "NodeName",
		},
	}
}

func (f *fixture) newController() (*Controller, kubeinformers.SharedInformerFactory) {
	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)

	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriodFunc())
	c := New(f.kubeclient, k8sI.Core().V1().Pods(), "NodeName")

	c.podsSynced = alwaysReady

	for _, p := range f.podsLister {
		k8sI.Core().V1().Pods().Informer().GetIndexer().Add(p)
	}

	return c, k8sI
}

type testBody func(c *Controller, k8si kubeinformers.SharedInformerFactory)

func (f *fixture) run(t testBody) {
	c, k8sI := f.newController()

	stopCh := make(chan struct{})
	defer close(stopCh)
	k8sI.Start(stopCh)

	t(c, k8sI)
}

func TestPublishesMetric(t *testing.T) {
	f := newFixture(t)
	pod := newPod("podname", "namespace", `[{
		"name": "kindnet",
		"interface": "eth0",
		"ips": [
			"10.244.0.10"
		],
		"mac": "4a:e9:0b:e2:63:67",
		"default": true,
		"dns": {}
	}]`)
	f.podsLister = append(f.podsLister, pod)
	f.kubeobjects = append(f.kubeobjects, pod)
	f.expectedMetrics = `
	pod_network_name_info{interface="eth0",namespace="namespace",network_name="kindnet",pod="podname"} 0
	`

	f.run(func(c *Controller, k8si kubeinformers.SharedInformerFactory) {
		c.podHandler(getKey(pod, t))
	})

	err := promtestutil.CollectAndCompare(podmetrics.NetAttachDefPerPod, strings.NewReader(metadata+f.expectedMetrics))
	if err != nil {
		t.Error("Failed to collect metrics", err)
	}
	podmetrics.NetAttachDefPerPod.Reset()
}

func TestDeletesMetric(t *testing.T) {
	f := newFixture(t)
	pod := newPod("podname", "namespace", `[{
		"name": "kindnet",
		"interface": "eth0",
		"ips": [
			"10.244.0.10"
		],
		"mac": "4a:e9:0b:e2:63:67",
		"default": true,
		"dns": {}
	}]`)
	f.podsLister = append(f.podsLister, pod)
	f.kubeobjects = append(f.kubeobjects, pod)
	f.expectedMetrics = `
	`

	f.run(func(c *Controller, k8si kubeinformers.SharedInformerFactory) {
		// send pod, then make it disappear simulating a delete
		c.podHandler(getKey(pod, t))
		f.podsLister = []*v1.Pod{}
		f.kubeobjects = []runtime.Object{}
		indxr := k8si.Core().V1().Pods().Informer().GetIndexer()
		for _, p := range indxr.List() {
			indxr.Delete(p)
		}
		c.podHandler(getKey(pod, t))
	})

	err := promtestutil.CollectAndCompare(podmetrics.NetAttachDefPerPod, strings.NewReader(metadata+f.expectedMetrics))
	if err != nil {
		t.Error("Failed to collect metrics", err)
	}
	podmetrics.NetAttachDefPerPod.Reset()
}

func getKey(pod *v1.Pod, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(pod)
	if err != nil {
		t.Errorf("Unexpected error getting key for foo %v: %v", pod.Name, err)
		return ""
	}
	return key
}
