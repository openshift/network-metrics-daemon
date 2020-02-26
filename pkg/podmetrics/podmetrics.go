package podmetrics

import (
	"net/http"
	"sync"

	"github.com/openshift/network-metrics-daemon/pkg/podnetwork"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog"
)

const (
	metricStoreInitSize int = 330
	initialMetricsCount int = 0
	metricsIncVal       int = 1
)

type podKey struct {
	name      string
	namespace string
}

var podNetworks = make(map[podKey][]podnetwork.Network)
var mtx sync.Mutex

var (
	// NetAttachDefPerPod represent the network attachment definitions bound to a given
	// pod
	NetAttachDefPerPod = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "network_attachment_definition_per_pod",
			Help: "Metric to identify clusters with network attachment definition enabled instances.",
		}, []string{"metricspod",
			"metricsnamespace",
			"interface",
			"networkname"})
)

//UpdateForPod adds metrics for all the provided networks to the given pod.
func UpdateForPod(podName, namespace string, networks []podnetwork.Network) {
	for _, n := range networks {
		if n.Interface == "" {
			// as we are interested in netlink interfaces
			// only, we are skipping networks with no interface
			continue
		}

		labels := prometheus.Labels{
			"metricspod":       podName,
			"metricsnamespace": namespace,
			"interface":        n.Interface,
			"networkname":      n.NetworkName,
		}
		NetAttachDefPerPod.With(labels).Add(0)
	}
	mtx.Lock()
	defer mtx.Unlock()
	podNetworks[podKey{podName, namespace}] = networks
}

// DeleteAllForPod stop publishing all the network metrics related to the
// given pod.
func DeleteAllForPod(podName, namespace string) {
	mtx.Lock()
	defer mtx.Unlock()
	nets, ok := podNetworks[podKey{podName, namespace}]
	if !ok {
		return
	}

	delete(podNetworks, podKey{podName, namespace})

	for _, n := range nets {
		labels := prometheus.Labels{
			"metricspod":       podName,
			"metricsnamespace": namespace,
			"interface":        n.Interface,
			"networkname":      n.NetworkName,
		}
		NetAttachDefPerPod.Delete(labels)
	}
}

// Serve serves the network metrics to the given address.
func Serve(metricsAddress string, stopCh <-chan struct{}) {

	// Including these stats kills performance when Prometheus polls with multiple targets
	prometheus.Unregister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	prometheus.Unregister(prometheus.NewGoCollector())

	prometheus.MustRegister(NetAttachDefPerPod)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	})

	klog.Info("Serving network metrics")
	server := &http.Server{Addr: metricsAddress, Handler: mux}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			klog.Error("Failed serving network metrics", err)

		}
	}()

	go func() {
		<-stopCh
		klog.Info("Received stop signal, closing the network metrics endpoint")
		server.Close()
	}()
}
