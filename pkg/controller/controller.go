package controller

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	"github.com/openshift/network-metrics-daemon/pkg/podmetrics"
	"github.com/openshift/network-metrics-daemon/pkg/podnetwork"
)

// Controller is the controller implementation for Foo resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	podsSynced    cache.InformerSynced
	indexer       cache.Indexer
	workqueue     workqueue.RateLimitingInterface
}

// New returns a new controller listening to pods.
func New(
	kubeclientset kubernetes.Interface,
	informer cache.SharedIndexInformer,
	currentNode string) *Controller {

	controller := &Controller{
		kubeclientset: kubeclientset,
		indexer:       informer.GetIndexer(),
		podsSynced:    informer.HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Pods"),
	}

	klog.Info("Setting up event handlers")

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*v1.Pod)
			_, ok := pod.Annotations[podnetwork.Status]
			if !ok {
				return
			}
			if pod.Spec.NodeName != currentNode {
				return
			}
			controller.enqueuePod(pod)
		},
		UpdateFunc: func(old, new interface{}) {
			newPod := new.(*v1.Pod)
			oldPod := old.(*v1.Pod)

			if newPod.Annotations[podnetwork.Status] == oldPod.Annotations[podnetwork.Status] {
				return
			}
			if newPod.Spec.NodeName != currentNode {
				return
			}
			controller.enqueuePod(new)
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*v1.Pod)
			if pod.Spec.NodeName != currentNode {
				return
			}
			controller.enqueuePod(pod)
		},
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting pod controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.podsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := c.podHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// podHandler receives a pod and updates the related pod network metrics
func (c *Controller) podHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	obj, exists, err := c.indexer.GetByKey(key)
	// Get the Pod resource with this namespace/name
	if err != nil {
		if errors.IsNotFound(err) {
			podmetrics.DeleteAllForPod(name, namespace)
			return nil
		}
		return err
	}

	if !exists {
		podmetrics.DeleteAllForPod(name, namespace)
		return nil
	}

	pod, ok := obj.(*v1.Pod)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("invalid object for key: %s", key))
		return nil
	}

	klog.Infof("Received pod '%s'", pod.Name)
	networks, err := podnetwork.Get(pod)
	if err != nil {
		return err
	}

	// As an interface might have been removed from the pod (or changed)
	// and eventually re-add them, as the chance of having the networks changed is
	// pretty low
	podmetrics.DeleteAllForPod(name, namespace)
	podmetrics.UpdateForPod(pod.Name, pod.Namespace, networks)
	return nil
}

func (c *Controller) enqueuePod(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}
