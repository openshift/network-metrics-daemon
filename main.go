package main

import (
	"flag"
	"os"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/openshift/network-metrics-daemon/pkg/controller"
	"github.com/openshift/network-metrics-daemon/pkg/podmetrics"
	"github.com/openshift/network-metrics-daemon/pkg/signals"
)

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
	flag.Parse()

	config.currentNode = os.Getenv("NODE_NAME")
	if config.currentNode == "" {
		klog.Fatalf("NODE_NAME required environment variable not set")
	}

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

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)

	ctrl := controller.New(kubeClient, kubeInformerFactory.Core().V1().Pods(), config.currentNode)
	kubeInformerFactory.Start(stopCh)

	podmetrics.Serve(config.metricsAddress, stopCh)

	if err = ctrl.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}
