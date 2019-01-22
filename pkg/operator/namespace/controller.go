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

package namespace

import (
	"reflect"
	"strconv"

	"github.com/golang/glog"
	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	clientset "github.com/inwinstack/blended/client/clientset/versioned"
	"github.com/inwinstack/ip-assigner/pkg/config"
	"github.com/inwinstack/ip-assigner/pkg/constants"
	"github.com/inwinstack/ip-assigner/pkg/k8sutil"
	opkit "github.com/inwinstack/operator-kit"
	slice "github.com/thoas/go-funk"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

var Resource = opkit.CustomResource{
	Name:    "namespace",
	Plural:  "namespaces",
	Version: "v1",
	Kind:    reflect.TypeOf(v1.Namespace{}).Name(),
}

type NamespaceController struct {
	ctx       *opkit.Context
	clientset clientset.Interface
	conf      *config.OperatorConfig
}

func NewController(ctx *opkit.Context, clientset clientset.Interface, conf *config.OperatorConfig) *NamespaceController {
	return &NamespaceController{ctx: ctx, clientset: clientset, conf: conf}
}

func (c *NamespaceController) StartWatch(namespace string, stopCh chan struct{}) error {
	resourceHandlerFuncs := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
	}

	glog.Infof("Start watching namespace resources.")
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.ctx.Clientset.CoreV1().RESTClient())
	go watcher.Watch(&v1.Namespace{}, stopCh)
	return nil
}

func (c *NamespaceController) onAdd(obj interface{}) {
	ns := obj.(*v1.Namespace).DeepCopy()
	glog.V(2).Infof("Received add on Namespace %s.", ns.Name)

	if !slice.ContainsString(c.conf.IgnoreNamespaces, ns.Name) {
		if err := c.makeAnntations(ns); err != nil {
			glog.Errorf("Failed to init %s namespace: %+v.", ns.Name, err)
		}

		if ns.Status.Phase != v1.NamespaceTerminating {
			if err := c.createOrDeleteIPs(ns); err != nil {
				glog.Errorf("Failed to create IPs in %s namespace: %+v.", ns.Name, err)
			}
		}
	}
}

func (c *NamespaceController) onUpdate(oldObj, newObj interface{}) {
	old := oldObj.(*v1.Namespace).DeepCopy()
	new := newObj.(*v1.Namespace).DeepCopy()
	glog.V(2).Infof("Received update on Namespace %s.", new.Name)

	if new.Status.Phase != v1.NamespaceTerminating {
		if err := c.createOrDeleteIPs(new); err != nil {
			glog.Errorf("Failed to create IPs in %s namespace: %+v.", new.Name, err)
		}
	}

	oldPool := old.Annotations[constants.AnnKeyPoolName]
	newPool := new.Annotations[constants.AnnKeyPoolName]
	if oldPool != newPool && oldPool != "" {
		// Cleanup old pool IPs
		old.Annotations[constants.AnnKeyNumberOfIP] = "0"
		old.Annotations[constants.AnnKeyDirtyResource] = "true"
		if err := c.createOrDeleteIPs(old); err != nil {
			glog.Errorf("Failed to cleanup IPs in %s namespace: %+v.", old.Name, err)
		}
	}
}

func (c *NamespaceController) makeAnntations(ns *v1.Namespace) error {
	if ns.Annotations == nil {
		ns.Annotations = map[string]string{}
	}

	if ns.Annotations[constants.AnnKeyNumberOfIP] == "" {
		ns.Annotations[constants.AnnKeyNumberOfIP] = strconv.Itoa(constants.DefaultNumberOfIP)
	}

	if ns.Annotations[constants.AnnKeyPoolName] == "" {
		ns.Annotations[constants.AnnKeyPoolName] = constants.DefaultPool
	}

	if _, err := c.ctx.Clientset.CoreV1().Namespaces().Update(ns); err != nil {
		return err
	}
	return nil
}

func (c *NamespaceController) createOrDeleteIPs(ns *v1.Namespace) error {
	pool, err := c.getPool(ns)
	if err != nil {
		return err
	}

	if slice.ContainsString(pool.Spec.IgnoreNamespaces, ns.Name) || !pool.Spec.AssignToNamespace {
		return nil
	}

	ips, err := c.clientset.InwinstackV1().IPs(ns.Name).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	ipNumber, err := strconv.Atoi(ns.Annotations[constants.AnnKeyNumberOfIP])
	if err != nil {
		return err
	}

	k8sutil.FilterIPsByPool(ips, pool)
	for i := 0; i < (ipNumber - len(ips.Items)); i++ {
		ip := k8sutil.NewIPWithNamespace(ns, pool.Name)
		if _, err := c.clientset.InwinstackV1().IPs(ns.Name).Create(ip); err != nil {
			return err
		}
	}

	for i := 0; i < (len(ips.Items) - ipNumber); i++ {
		ip := ips.Items[len(ips.Items)-(1+i)]
		c.markIPDirty(&ip)
		if _, err := c.clientset.InwinstackV1().IPs(ns.Name).Update(&ip); err != nil {
			return err
		}
	}
	return nil
}

func (c *NamespaceController) getPool(ns *v1.Namespace) (*inwinv1.Pool, error) {
	poolName := ns.Annotations[constants.AnnKeyPoolName]
	pool, err := c.clientset.InwinstackV1().Pools().Get(poolName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func (c *NamespaceController) markIPDirty(ip *inwinv1.IP) {
	if ip.Annotations == nil {
		ip.Annotations = map[string]string{}
	}
	ip.Annotations[constants.AnnKeyDirtyResource] = "true"
}
