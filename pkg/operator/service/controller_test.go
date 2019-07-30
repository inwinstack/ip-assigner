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
	"testing"
	"time"

	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	blendedfake "github.com/inwinstack/blended/generated/clientset/versioned/fake"
	"github.com/inwinstack/ip-assigner/pkg/config"
	"github.com/inwinstack/ip-assigner/pkg/constants"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

const timeout = time.Second * 3

func TestServiceController(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Config{
		Threads:    2,
		PublicPool: "internet",
	}

	clientset := fake.NewSimpleClientset()
	blendedset := blendedfake.NewSimpleClientset()
	informer := informers.NewSharedInformerFactory(clientset, 0)

	controller := NewController(cfg, clientset, blendedset, informer.Core().V1().Services())
	go informer.Start(ctx.Done())
	assert.Nil(t, controller.Run(ctx, cfg.Threads))

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test1"},
	}
	_, nserr := clientset.CoreV1().Namespaces().Create(ns)
	assert.Nil(t, nserr)

	ip := &blendedv1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "172.11.22.33",
			Namespace: ns.Name,
		},
		Spec: blendedv1.IPSpec{
			PoolName: cfg.PublicPool,
		},
		Status: blendedv1.IPStatus{
			Phase:   blendedv1.IPActive,
			Address: "140.11.22.33",
		},
	}
	_, iperr := blendedset.InwinstackV1().IPs(ip.Namespace).Create(ip)
	assert.Nil(t, iperr)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: ns.Name,
			Annotations: map[string]string{
				constants.PublicPoolKey: cfg.PublicPool,
			},
		},
		Spec: corev1.ServiceSpec{
			ExternalIPs: []string{"172.11.22.33"},
			Type:        corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}
	_, svcerr := clientset.CoreV1().Services(svc.Namespace).Create(svc)
	assert.Nil(t, svcerr)

	// Check IP address
	failed := true
	for start := time.Now(); time.Since(start) < timeout; {
		svc, err := clientset.CoreV1().Services(ns.Name).Get(svc.Name, metav1.GetOptions{})
		assert.Nil(t, err)
		if address, ok := svc.Annotations[constants.PublicIPKey]; ok {
			assert.Equal(t, ip.Status.Address, address)
			failed = false
			break
		}
	}
	assert.Equal(t, false, failed, "cannot get public IP.")

	// Test for deleting
	newSvc, _ := clientset.CoreV1().Services(ns.Name).Get(svc.Name, metav1.GetOptions{})
	assert.Nil(t, clientset.CoreV1().Services(ns.Name).Delete(svc.Name, nil))

	controller.cleanup(newSvc)

	ipList, err := blendedset.InwinstackV1().IPs(ns.Name).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(ipList.Items))

	cancel()
	controller.Stop()
}
