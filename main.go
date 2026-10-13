package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/openshift/network-metrics-daemon/pkg/controller"
	"github.com/openshift/network-metrics-daemon/pkg/podmetrics"
	"github.com/openshift/network-metrics-daemon/pkg/signals"
)

// build is the git version of this program. It is set using build flags in the makefile.
var build = "develop"

func main() {
	klog.InitFlags(nil)
	var config struct {
		kubeconfig     string
		masterURL      string
		metricsAddress string
		currentNode    string
	}

	flag.StringVar(&config.kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&config.masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&config.metricsAddress, "metrics-listen-address", ":9091", "metrics server listen address.")
	flag.StringVar(&config.currentNode, "node-name", "", "the node the daemon is running on.")

	flag.Parse()

	if config.currentNode == "" {
		klog.Fatalf("--node-name required parameter not set")
	}
	klog.Info("Version:", build)
	klog.Info("Starting with config", config)

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(config.masterURL, config.kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	fieldSelector := fmt.Sprintf("spec.nodeName=%s", config.currentNode)
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fieldSelector
				return kubeClient.CoreV1().Pods(metav1.NamespaceAll).List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = fieldSelector
				return kubeClient.CoreV1().Pods(metav1.NamespaceAll).Watch(context.Background(), options)
			},
		},
		&v1.Pod{},
		time.Second*30,
		cache.Indexers{},
	)

	ctrl := controller.New(kubeClient, informer, config.currentNode)
	go informer.Run(stopCh)

	podmetrics.Serve(config.metricsAddress, stopCh)

	if err = ctrl.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}
