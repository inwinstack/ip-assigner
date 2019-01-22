/*
Copyright © 2018 inwinSTACK.inc

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

package ip

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/glog"
	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	clientset "github.com/inwinstack/blended/client/clientset/versioned"
	"github.com/inwinstack/ip-assigner/pkg/constants"
	opkit "github.com/inwinstack/operator-kit"
	slice "github.com/thoas/go-funk"
	"k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/tools/cache"
)

const (
	customResourceName       = "ip"
	customResourceNamePlural = "ips"
	updateAction             = "update"
	deleteAction             = "delete"
)

var Resource = opkit.CustomResource{
	Name:    customResourceName,
	Plural:  customResourceNamePlural,
	Group:   inwinv1.CustomResourceGroup,
	Version: inwinv1.Version,
	Scope:   apiextensionsv1beta1.NamespaceScoped,
	Kind:    reflect.TypeOf(inwinv1.IP{}).Name(),
}

type IPController struct {
	ctx       *opkit.Context
	clientset clientset.Interface
}

func NewController(ctx *opkit.Context, clientset clientset.Interface) *IPController {
	return &IPController{ctx: ctx, clientset: clientset}
}

func (c *IPController) StartWatch(namespace string, stopCh chan struct{}) error {
	resourceHandlerFuncs := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}

	glog.Infof("Start watching IP resources.")
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.clientset.InwinstackV1().RESTClient())
	go watcher.Watch(&inwinv1.IP{}, stopCh)
	return nil
}

func (c *IPController) onAdd(obj interface{}) {
	ip := obj.(*inwinv1.IP).DeepCopy()
	glog.V(2).Infof("Received add on IP %s in %s namespace.", ip.Name, ip.Namespace)

	if err := c.syncIP(ip); err != nil {
		glog.Errorf("%+v.", err)
	}
}

func (c *IPController) onUpdate(oldObj, newObj interface{}) {
	ip := newObj.(*inwinv1.IP).DeepCopy()
	glog.V(2).Infof("Received update on IP %s in %s namespace.", ip.Name, ip.Namespace)

	if err := c.syncIP(ip); err != nil {
		glog.Errorf("%+v.", err)
	}
}

func (c *IPController) onDelete(obj interface{}) {
	ip := obj.(*inwinv1.IP).DeepCopy()
	glog.V(2).Infof("Received delete on IP %s in %s namespace.", ip.Name, ip.Namespace)

	if err := c.updateNamespaceIPs(deleteAction, ip); err != nil {
		glog.Errorf("Failed to sync namespace annotations on IP %s in %s namespace: %+v.", ip.Name, ip.Namespace, err)
	}
}

func (c *IPController) getNamespace(ip *inwinv1.IP) (*v1.Namespace, error) {
	return c.ctx.Clientset.CoreV1().Namespaces().Get(ip.Namespace, metav1.GetOptions{})
}

func (c *IPController) getPool(ns *v1.Namespace) (*inwinv1.Pool, error) {
	poolName := ns.Annotations[constants.AnnKeyPoolName]
	return c.clientset.InwinstackV1().Pools().Get(poolName, metav1.GetOptions{})
}

func (c *IPController) syncIP(ip *inwinv1.IP) error {
	if _, dirty := ip.Annotations[constants.AnnKeyDirtyResource]; dirty {
		if err := c.cleanupDirtyIP(ip); err != nil {
			return fmt.Errorf("Failed to cleanup dirty on IP %s in %s namespace: %+v", ip.Name, ip.Namespace, err)
		}
		return nil
	}

	if err := c.updateNamespaceIPs(updateAction, ip); err != nil {
		return fmt.Errorf("Failed to sync namespace annotations on IP %s in %s namespace: %+v", ip.Name, ip.Namespace, err)
	}
	return nil
}

func (c *IPController) updateNamespaceIPs(action string, ip *inwinv1.IP) error {
	ns, err := c.getNamespace(ip)
	if err != nil {
		return err
	}

	// If Namespace annotation is empty, then ignore this Namespace.
	if _, ok := ns.Annotations[constants.AnnKeyPoolName]; !ok {
		return nil
	}

	pool, err := c.getPool(ns)
	if err != nil {
		return err
	}

	ignoreNamespace := slice.ContainsString(pool.Spec.IgnoreNamespaces, ns.Name)
	ignorePool := (ip.Spec.PoolName != pool.Name)
	_, dirty := ip.Annotations[constants.AnnKeyDirtyResource]
	if (ignoreNamespace || ignorePool || pool.Spec.IgnoreNamespaceAnnotation) && !dirty {
		return nil
	}

	if ns.Status.Phase == v1.NamespaceActive {
		if ip.Status.Address != "" {
			var ips []string
			if nsIPs := ns.Annotations[constants.AnnKeyIPs]; nsIPs != "" {
				ips = strings.Split(nsIPs, ",")
			}

			switch action {
			case updateAction:
				ips = append(ips, ip.Status.Address)
				ips = slice.UniqString(ips)
			case deleteAction:
				ips = slice.FilterString(ips, func(v string) bool {
					return v != ip.Status.Address
				})
			}

			ns.Annotations[constants.AnnKeyIPs] = strings.Join(ips, ",")
			if len(ips) > 0 {
				ns.Annotations[constants.AnnKeyLatestIP] = ips[len(ips)-1]
			}

			if len(ips) == 0 {
				delete(ns.Annotations, constants.AnnKeyIPs)
				delete(ns.Annotations, constants.AnnKeyLatestIP)
			}
		}

		if _, err := c.ctx.Clientset.CoreV1().Namespaces().Update(ns); err != nil {
			return err
		}
	}
	return nil
}

func (c *IPController) cleanupDirtyIP(ip *inwinv1.IP) error {
	if err := c.clientset.InwinstackV1().IPs(ip.Namespace).Delete(ip.Name, nil); err != nil {
		return err
	}
	return nil
}
