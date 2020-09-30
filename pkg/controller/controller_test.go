package controller

import (
	"context"
	"strings"
	"testing"
	"time"

	promtestutil "github.com/prometheus/client_golang/prometheus/testutil"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
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

func (f *fixture) newController() (*Controller, cache.SharedInformer) {
	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return f.kubeclient.CoreV1().Pods(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return f.kubeclient.CoreV1().Pods(metav1.NamespaceAll).Watch(context.Background(), metav1.ListOptions{})
			},
		},
		&v1.Pod{},
		0, //Skip resync
		cache.Indexers{},
	)
	c := New(f.kubeclient, informer, "NodeName")

	c.podsSynced = alwaysReady

	for _, p := range f.podsLister {
		informer.GetIndexer().Add(p)
	}

	return c, informer
}

type testBody func(c *Controller, informer cache.SharedInformer)

func (f *fixture) run(t testBody) {
	c, informer := f.newController()

	stopCh := make(chan struct{})
	defer close(stopCh)
	go informer.Run(stopCh)

	t(c, informer)
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

	f.run(func(c *Controller, informer cache.SharedInformer) {
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

	f.run(func(c *Controller, informer cache.SharedInformer) {
		// send pod, then make it disappear simulating a delete
		c.podHandler(getKey(pod, t))
		f.podsLister = []*v1.Pod{}
		f.kubeobjects = []runtime.Object{}
		indxr := informer.GetStore()
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
