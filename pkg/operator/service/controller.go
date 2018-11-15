package service

import (
	"reflect"

	"github.com/golang/glog"
	inwinclientset "github.com/inwinstack/blended/client/clientset/versioned/typed/inwinstack/v1"
	opkit "github.com/inwinstack/operator-kit"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

var Resource = opkit.CustomResource{
	Name:    "service",
	Plural:  "services",
	Version: "v1",
	Kind:    reflect.TypeOf(v1.Service{}).Name(),
}

type Controller struct {
	ctx       *opkit.Context
	clientset inwinclientset.InwinstackV1Interface
}

func NewController(ctx *opkit.Context, clientset inwinclientset.InwinstackV1Interface) *Controller {
	return &Controller{ctx: ctx, clientset: clientset}
}

func (c *Controller) StartWatch(namespace string, stopCh chan struct{}) error {
	resourceHandlerFuncs := cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}

	glog.Info("Start watching service resources.")
	watcher := opkit.NewWatcher(Resource, namespace, resourceHandlerFuncs, c.ctx.Clientset.CoreV1().RESTClient())
	go watcher.Watch(&v1.Service{}, stopCh)
	return nil
}

func (c *Controller) onAdd(obj interface{}) {
	svc := obj.(*v1.Service).DeepCopy()
	glog.V(2).Infof("Received add on Service %s in %s namespace.", svc.Name, svc.Namespace)
}

func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	// old := oldObj.(*v1.Service).DeepCopy()
	svc := newObj.(*v1.Service).DeepCopy()
	glog.V(2).Infof("Received update on Service %s in %s namespace.", svc.Name, svc.Namespace)

}

func (c *Controller) onDelete(obj interface{}) {
	svc := obj.(*v1.Service).DeepCopy()
	glog.V(2).Infof("Received delete on Service %s in %s namespace.", svc.Name, svc.Namespace)
}
