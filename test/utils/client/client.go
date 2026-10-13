package client

import (
	"os"

	"github.com/golang/glog"

	k8scnicncfiov1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned/typed/k8s.cni.cncf.io/v1"
	"k8s.io/client-go/discovery"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client defines the client set that will be used for testing
var Client *APISet

func init() {
	Client = New("")
}

// APISet provides the struct to talk with relevant API
type APISet struct {
	corev1.CoreV1Interface
	appsv1.AppsV1Interface
	discovery.DiscoveryInterface
	k8scnicncfiov1.K8sCniCncfIoV1Interface
	Config *rest.Config
}

// New returns a *ClientSet for the given kubeconfig.
func New(kubeconfig string) *APISet {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}

	if kubeconfig != "" {
		glog.V(4).Infof("Loading kube client config from path %q", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		glog.V(4).Infof("Using in-cluster kube client config")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		glog.Infof("Failed to create a valid client")
		return nil
	}

	clientSet := &APISet{}
	clientSet.CoreV1Interface = corev1.NewForConfigOrDie(config)
	clientSet.AppsV1Interface = appsv1.NewForConfigOrDie(config)
	clientSet.DiscoveryInterface = discovery.NewDiscoveryClientForConfigOrDie(config)
	clientSet.K8sCniCncfIoV1Interface = k8scnicncfiov1.NewForConfigOrDie(config)
	clientSet.Config = config

	return clientSet
}
