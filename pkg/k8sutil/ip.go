package k8sutil

import (
	"reflect"

	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewIP(name, namespace, pool string, owner *v1.Service) *inwinv1.IP {
	return &inwinv1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(owner, schema.GroupVersionKind{
					Group:   v1.SchemeGroupVersion.Group,
					Version: v1.SchemeGroupVersion.Version,
					Kind:    reflect.TypeOf(v1.Service{}).Name(),
				}),
			},
		},
		Spec: inwinv1.IPSpec{
			PoolName:        pool,
			UpdateNamespace: false,
		},
	}
}
