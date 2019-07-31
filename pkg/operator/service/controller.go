/*
Copyright © 2018 inwinSTACK Inc

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package service

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/golang/glog"
	blended "github.com/inwinstack/blended/generated/clientset/versioned"
	blended_k8sutil "github.com/inwinstack/blended/k8sutil"
	"github.com/inwinstack/ip-assigner/pkg/config"
	"github.com/inwinstack/ip-assigner/pkg/constants"
	"github.com/inwinstack/ip-assigner/pkg/k8sutil"
	"github.com/thoas/go-funk"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informerv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller represents the controller of service
type Controller struct {
	clientset  kubernetes.Interface
	blendedset blended.Interface
	lister     listerv1.ServiceLister
	synced     cache.InformerSynced
	queue      workqueue.RateLimitingInterface
	cfg        *config.Config
}

// NewController creates an instance of the service controller
func NewController(
	cfg *config.Config,
	clientset kubernetes.Interface,
	blendedset blended.Interface,
	informer informerv1.ServiceInformer) *Controller {
	controller := &Controller{
		cfg:        cfg,
		clientset:  clientset,
		blendedset: blendedset,
		lister:     informer.Lister(),
		synced:     informer.Informer().HasSynced,
		queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Services"),
	}
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueue,
		UpdateFunc: func(old, new interface{}) {
			oo := old.(*v1.Service)
			no := new.(*v1.Service)
			ooPool := oo.Annotations[constants.PublicPoolKey]
			noPool := no.Annotations[constants.PublicPoolKey]
			if ooPool != noPool {
				// Cannot change the pool name
				no.Annotations[constants.PublicPoolKey] = ooPool
			}
			controller.enqueue(no)
		},
	})
	return controller
}

// Run serves the service controller
func (c *Controller) Run(ctx context.Context, threadiness int) error {
	glog.Info("Starting Service controller")
	glog.Info("Waiting for Service informer caches to sync")
	if ok := cache.WaitForCacheSync(ctx.Done(), c.synced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, ctx.Done())
	}
	return nil
}

// Stop stops the namespace controller
func (c *Controller) Stop() {
	glog.Info("Stopping Service controller")
	c.queue.ShutDown()
}

func (c *Controller) runWorker() {
	defer utilruntime.HandleCrash()
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.queue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.queue.Done(obj)
		key, ok := obj.(string)
		if !ok {
			c.queue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("Service controller expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.reconcile(key); err != nil {
			c.queue.AddRateLimited(key)
			return fmt.Errorf("Service controller error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.queue.Forget(obj)
		glog.V(2).Infof("Service controller successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.queue.Add(key)
}

func (c *Controller) reconcile(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return err
	}

	svc, err := c.lister.Services(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("service '%s' in work queue no longer exists", key))
			return err
		}
		return err
	}

	// If service was deleted, it will clean up IP.
	if !svc.ObjectMeta.DeletionTimestamp.IsZero() {
		return c.cleanup(svc)
	}

	c.makeDefaultPool(svc)
	if len(svc.Spec.ExternalIPs) == 0 {
		return nil
	}

	if err := c.allocate(svc); err != nil {
		return err
	}

	address := net.ParseIP(svc.Annotations[constants.PublicIPKey])
	if address == nil {
		return fmt.Errorf("failed to get the public IP")
	}

	svcCopy := svc.DeepCopy()
	if !funk.ContainsString(svcCopy.Finalizers, constants.Finalizer) {
		blended_k8sutil.AddFinalizer(&svcCopy.ObjectMeta, constants.Finalizer)
	}

	if _, err := c.clientset.CoreV1().Services(svcCopy.Namespace).Update(svcCopy); err != nil {
		return err
	}
	return nil
}

func (c *Controller) makeDefaultPool(svc *v1.Service) {
	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}

	if _, ok := svc.Annotations[constants.PublicPoolKey]; !ok {
		svc.Annotations[constants.PublicPoolKey] = c.cfg.PublicPool
	}
}

func (c *Controller) allocate(svc *v1.Service) error {
	pool := svc.Annotations[constants.PublicPoolKey]
	address := net.ParseIP(svc.Annotations[constants.PublicIPKey])
	if address == nil && len(pool) > 0 {
		name := svc.Spec.ExternalIPs[0]
		ip, err := c.blendedset.InwinstackV1().IPs(svc.Namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			if net.ParseIP(ip.Status.Address) != nil {
				svc.Annotations[constants.PublicIPKey] = ip.Status.Address
			}
			return nil
		}

		if _, err := k8sutil.NewIP(c.blendedset, name, svc.Namespace, pool); err != nil {
			return err
		}
		return fmt.Errorf("public IP has been allocated, but cannot get")
	}
	return nil
}

func (c *Controller) deallocate(svc *v1.Service) error {
	name := svc.Spec.ExternalIPs[0]
	if _, err := c.blendedset.InwinstackV1().IPs(svc.Namespace).Get(name, metav1.GetOptions{}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return c.blendedset.InwinstackV1().IPs(svc.Namespace).Delete(name, nil)
}

func (c *Controller) cleanup(svc *v1.Service) error {
	svcCopy := svc.DeepCopy()
	address := net.ParseIP(svcCopy.Annotations[constants.PublicIPKey])
	if address == nil {
		return nil
	}

	svcs, err := c.clientset.CoreV1().Services(svcCopy.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	items := funk.Filter(svcs.Items, func(s v1.Service) bool {
		v := s.Annotations[constants.PublicIPKey]
		return v == address.String()
	})

	// If this namespace has other services are used the same public IP,
	// it will not release this public IP
	if len(items.([]v1.Service)) > 1 {
		if err := c.removeFinalizer(svcCopy); err != nil {
			return err
		}
		return nil
	}

	if err := c.deallocate(svcCopy); err != nil {
		return err
	}
	glog.V(3).Infof("Service controller has been deleted IP.")
	return c.removeFinalizer(svcCopy)
}

func (c *Controller) removeFinalizer(svc *v1.Service) error {
	blended_k8sutil.RemoveFinalizer(&svc.ObjectMeta, constants.Finalizer)
	if _, err := c.clientset.CoreV1().Services(svc.Namespace).Update(svc); err != nil {
		return err
	}
	return nil
}
