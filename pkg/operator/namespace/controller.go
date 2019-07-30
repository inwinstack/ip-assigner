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

package namespace

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	blended "github.com/inwinstack/blended/generated/clientset/versioned"
	"github.com/inwinstack/ip-assigner/pkg/config"
	"github.com/inwinstack/ip-assigner/pkg/constants"
	"github.com/inwinstack/ip-assigner/pkg/k8sutil"
	"github.com/thoas/go-funk"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	informerv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller represents the controller of namespace
type Controller struct {
	cfg *config.Config

	clientset  kubernetes.Interface
	blendedset blended.Interface
	lister     listerv1.NamespaceLister
	synced     cache.InformerSynced
	queue      workqueue.RateLimitingInterface
}

// NewController creates an instance of the namespace controller
func NewController(
	cfg *config.Config,
	clientset kubernetes.Interface,
	blendedset blended.Interface,
	informer informerv1.NamespaceInformer) *Controller {
	controller := &Controller{
		cfg:        cfg,
		clientset:  clientset,
		blendedset: blendedset,
		lister:     informer.Lister(),
		synced:     informer.Informer().HasSynced,
		queue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Namespaces"),
	}
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueue,
		UpdateFunc: func(old, new interface{}) {
			oo := old.(*v1.Namespace)
			no := new.(*v1.Namespace)
			ooPool := oo.Annotations[constants.PrivatePoolKey]
			noPool := no.Annotations[constants.PrivatePoolKey]
			if ooPool != "" && noPool != "" && ooPool != noPool {
				no.Annotations[constants.LatestPoolKey] = ooPool
			}
			controller.enqueue(no)
		},
	})
	return controller
}

// Run serves the namespace controller
func (c *Controller) Run(ctx context.Context, threadiness int) error {
	glog.Info("Starting Namespace controller")
	glog.Info("Waiting for Namespace informer caches to sync")
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
	glog.Info("Stopping the Namespace controller")
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
			utilruntime.HandleError(fmt.Errorf("Namespace controller expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.reconcile(key); err != nil {
			c.queue.AddRateLimited(key)
			return fmt.Errorf("Namespace controller error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.queue.Forget(obj)
		glog.V(2).Infof("Namespace controller successfully synced '%s'", key)
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
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return err
	}

	ns, err := c.lister.Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("namespace '%s' in work queue no longer exists", key))
			return err
		}
		return err
	}

	c.makeDefaultPool(ns)
	pool, err := k8sutil.GetPool(c.blendedset, ns.ObjectMeta, constants.PrivatePoolKey)
	if err != nil {
		return err
	}

	if funk.ContainsString(pool.Spec.IgnoreNamespaces, ns.Name) || !pool.Spec.AssignToNamespace {
		return nil
	}

	if _, ok := ns.Annotations[constants.LatestPoolKey]; ok {
		if err := c.cleanUpLatestIPs(ns); err != nil {
			return err
		}
	}

	if err := c.syncIPs(ns, pool.Name); err != nil {
		return err
	}
	return c.updateStatus(ns, pool.Name)
}

func (c *Controller) makeDefaultPool(ns *v1.Namespace) {
	if ns.Annotations == nil {
		ns.Annotations = map[string]string{}
	}

	if ns.Annotations[constants.NumberOfIPKey] == "" {
		ns.Annotations[constants.NumberOfIPKey] = strconv.Itoa(constants.DefaultNumberOfIP)
	}

	if _, err := strconv.Atoi(ns.Annotations[constants.NumberOfIPKey]); err != nil {
		ns.Annotations[constants.NumberOfIPKey] = strconv.Itoa(constants.DefaultNumberOfIP)
	}

	if ns.Annotations[constants.PrivatePoolKey] == "" {
		ns.Annotations[constants.PrivatePoolKey] = c.cfg.PrivatePool
	}
}

func (c *Controller) syncIPs(ns *v1.Namespace, poolName string) error {
	ips, err := c.blendedset.InwinstackV1().IPs(ns.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(ns.Annotations[constants.NumberOfIPKey])
	if err != nil {
		return err
	}

	k8sutil.FilterIPsByPool(ips, poolName)
	sort.Slice(ips.Items, func(i, j int) bool {
		return ips.Items[i].Status.LastUpdateTime.Time.Before(ips.Items[j].Status.LastUpdateTime.Time)
	})
	return c.createOrDeleteIPs(ns, ips, number, poolName)
}

func (c *Controller) cleanUpLatestIPs(ns *v1.Namespace) error {
	ips, err := c.blendedset.InwinstackV1().IPs(ns.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	poolName := ns.Annotations[constants.LatestPoolKey]
	k8sutil.FilterIPsByPool(ips, poolName)
	if err := c.createOrDeleteIPs(ns, ips, 0, poolName); err != nil {
		return err
	}
	delete(ns.Annotations, constants.LatestPoolKey)
	return nil
}

func (c *Controller) createOrDeleteIPs(ns *v1.Namespace, ips *blendedv1.IPList, number int, poolName string) error {
	// Create IPs if the number is more than the length of ips.Items.
	for i := 0; i < (number - len(ips.Items)); i++ {
		name := fmt.Sprintf("%s", uuid.NewUUID())
		if _, err := k8sutil.NewIP(c.blendedset, name, ns.Name, poolName); err != nil {
			return err
		}
	}

	// Delete IPs if the number is less than the length of ips.Items.
	for i := 0; i < (len(ips.Items) - number); i++ {
		ip := ips.Items[len(ips.Items)-(1+i)]
		if err := c.blendedset.InwinstackV1().IPs(ns.Name).Delete(ip.Name, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) updateStatus(ns *v1.Namespace, poolName string) error {
	nsCopy := ns.DeepCopy()
	ips, err := c.blendedset.InwinstackV1().IPs(nsCopy.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	number, err := strconv.Atoi(nsCopy.Annotations[constants.NumberOfIPKey])
	if err != nil {
		return err
	}

	switch {
	case number == 0:
		delete(nsCopy.Annotations, constants.LatestIPKey)
		delete(nsCopy.Annotations, constants.IPsKey)
	case number > 0:
		k8sutil.FilterIPsByPool(ips, poolName)
		sort.Slice(ips.Items, func(i, j int) bool {
			return ips.Items[i].Status.LastUpdateTime.Time.Before(ips.Items[j].Status.LastUpdateTime.Time)
		})

		var addrs []string
		for _, ip := range ips.Items {
			if ip.ObjectMeta.DeletionTimestamp.IsZero() {
				if ip.Status.Phase == blendedv1.IPFailed {
					continue
				}

				addr := net.ParseIP(ip.Status.Address)
				if addr == nil {
					return fmt.Errorf("failed to get IP address")
				}
				addrs = append(addrs, addr.String())
			}
		}

		nsCopy.Annotations[constants.IPsKey] = strings.Join(addrs, ",")
		if len(addrs) > 0 {
			nsCopy.Annotations[constants.LatestIPKey] = addrs[len(addrs)-1]
		}
	}

	if _, err := c.clientset.CoreV1().Namespaces().Update(nsCopy); err != nil {
		return err
	}
	return nil
}
