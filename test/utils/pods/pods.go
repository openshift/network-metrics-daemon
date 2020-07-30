package pods

import (
	"bytes"
	"os"
	"strings"

	"github.com/openshift/network-metrics-daemon/test/utils/client"
	"github.com/openshift/network-metrics-daemon/test/utils/consts"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/utils/pointer"
)

const hostnameLabel = "kubernetes.io/hostname"

// ExecCommand runs command in the pod and returns buffer output
func ExecCommand(cs *client.APISet, pod corev1.Pod, command []string) (bytes.Buffer, error) {
	var buf bytes.Buffer
	req := client.Client.CoreV1Interface.RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pod.Spec.Containers[0].Name,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(cs.Config, "POST", req.URL())
	if err != nil {
		return buf, err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: &buf,
		Stderr: os.Stderr,
		Tty:    true,
	})
	if err != nil {
		return buf, err
	}

	return buf, nil
}

func getDefinition() *corev1.Pod {
	podObject := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "testpod-",
			Namespace:    consts.TestingNamespace,
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: pointer.Int64Ptr(0),
			Containers: []corev1.Container{{Name: "test",
				Image:   "centos",
				Command: []string{"/bin/bash", "-c", "sleep INF"},
			}},
		},
	}

	return podObject
}

// DefineWithNetworks Defines a pod with networks
func DefineWithNetworks(networks []string) *corev1.Pod {
	podObject := getDefinition()
	podObject.Annotations = map[string]string{"k8s.v1.cni.cncf.io/networks": strings.Join(networks, ",")}

	return podObject
}

// RedefineWithCommand updates the pod defintion with a different command
func RedefineWithCommand(pod *corev1.Pod, command []string, args []string) *corev1.Pod {
	pod.Spec.Containers[0].Command = command
	pod.Spec.Containers[0].Args = args
	return pod
}

// RedefineWithRestartPolicy updates the pod defintion with a restart policy
func RedefineWithRestartPolicy(pod *corev1.Pod, restartPolicy corev1.RestartPolicy) *corev1.Pod {
	pod.Spec.RestartPolicy = restartPolicy
	return pod
}

// RedifineWithSpecificNode redifines a pod with a specific node
func RedifineWithSpecificNode(pod *corev1.Pod, nodeName string) *corev1.Pod {
	pod.Spec.NodeName = nodeName
	return pod
}
