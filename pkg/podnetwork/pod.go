package podnetwork

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// Status is the name of the network status annotation
const Status = "k8s.v1.cni.cncf.io/networks-status"

type status struct {
	Name      string `json:"name"`
	Interface string `json:"interface,omitempty"`
}

// Network represents the link between the pod,
// the interface name and the network attachment definition name
type Network struct {
	Interface   string
	NetworkName string
}

// Get return a slice of Networks info taken
// from the network status annotation of the given pod.
func Get(pod *corev1.Pod) ([]Network, error) {
	annotation, ok := pod.GetAnnotations()[Status]
	if !ok || annotation == "" {
		return make([]Network, 0), nil
	}

	var statuses []status
	if err := json.Unmarshal([]byte(annotation), &statuses); err != nil {
		return nil, fmt.Errorf("Failed to parse network status annotation for pod %s %v - [%s]", pod.Name, err, annotation)
	}

	res := make([]Network, len(statuses))
	for i, s := range statuses {
		res[i].Interface = s.Interface
		res[i].NetworkName = s.Name
	}
	return res, nil
}
