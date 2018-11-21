package service

import (
	"reflect"

	"github.com/golang/glog"
	inwinclientset "github.com/inwinstack/blended/client/clientset/versioned/typed/inwinstack/v1"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/kairen/ip-assigner/pkg/constants"
	"github.com/kairen/ip-assigner/pkg/k8sutil"
	"github.com/kairen/ip-assigner/pkg/util/slice"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/cache"
)

var Resource = opkit.CustomResource{
	Name:    "service",
	Plural:  "services",
	Version: "v1",
	Kind:    reflect.TypeOf(v1.Service{}).Name(),
}

type Controller struct {
	ctx              *opkit.Context
	clientset        inwinclientset.InwinstackV1Interface
	ignoreNamespaces []string
}

func NewController(ctx *opkit.Context, clientset inwinclientset.InwinstackV1Interface, ignoreNamespaces []string) *Controller {
	return &Controller{ctx: ctx, clientset: clientset, ignoreNamespaces: ignoreNamespaces}
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

	if err := c.sync(svc); err != nil {
		glog.Errorf("Failed to sync on Service %s in %s namespace: %+v.", svc.Name, svc.Namespace, err)
	}
}

func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	// old := oldObj.(*v1.Service).DeepCopy()
	svc := newObj.(*v1.Service).DeepCopy()
	glog.V(2).Infof("Received update on Service %s in %s namespace.", svc.Name, svc.Namespace)

	if svc.DeletionTimestamp == nil {
		if err := c.sync(svc); err != nil {
			glog.Errorf("Failed to sync on Service %s in %s namespace: %+v.", svc.Name, svc.Namespace, err)
		}
	}
}

func (c *Controller) onDelete(obj interface{}) {
	svc := obj.(*v1.Service).DeepCopy()
	glog.V(2).Infof("Received delete on Service %s in %s namespace.", svc.Name, svc.Namespace)
}

func (c *Controller) getPool(svc *v1.Service) string {
	if pool, ok := svc.Annotations[constants.AnnKeyAddressPool]; ok {
		return pool
	}
	return ""
}

func (c *Controller) makeRefresh(svc *v1.Service) {
	if len(svc.Spec.ExternalIPs) == 0 {
		if svc.Annotations == nil {
			svc.Annotations = map[string]string{}
		}
		svc.Annotations[constants.AnnKeyServiceRefresh] = string(uuid.NewUUID())
	}
}

func (c *Controller) sync(svc *v1.Service) error {
	if slice.ContainsStr(c.ignoreNamespaces, svc.Namespace) {
		return nil
	}

	pool := c.getPool(svc)
	if len(svc.Spec.Ports) == 0 || pool == "" {
		return nil
	}

	if err := c.allocateIP(svc); err != nil {
		glog.Errorf("Failed to allocate IP: %+v.", err)
	}

	c.makeRefresh(svc)
	if _, err := c.ctx.Clientset.CoreV1().Services(svc.Namespace).Update(svc); err != nil {
		return err
	}
	return nil
}

func (c *Controller) allocateIP(svc *v1.Service) error {
	pool := c.getPool(svc)
	if len(svc.Spec.ExternalIPs) == 0 {
		ip, err := c.clientset.IPs(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
		if err == nil {
			if ip.Status.Address != "" {
				delete(svc.Annotations, constants.AnnKeyServiceRefresh)
				svc.Spec.ExternalIPs = []string{ip.Status.Address}
				for _, port := range svc.Spec.Ports {
					ip.Status.Ports = append(ip.Status.Ports, int(port.Port))
				}

				if _, err := c.clientset.IPs(svc.Namespace).Update(ip); err != nil {
					return err
				}
			}
			return nil
		}

		newIP := k8sutil.NewIP(svc.Name, svc.Namespace, pool, svc)
		if _, err := c.clientset.IPs(svc.Namespace).Create(newIP); err != nil {
			return err
		}
	}
	return nil
}
