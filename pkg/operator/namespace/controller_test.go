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
	"strconv"
	"testing"
	"time"

	"github.com/inwinstack/ip-assigner/pkg/constants"

	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	blendedfake "github.com/inwinstack/blended/generated/clientset/versioned/fake"
	"github.com/inwinstack/ip-assigner/pkg/config"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

const timeout = time.Second * 3

func TestNamespaceController(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := &config.Config{
		Threads:    2,
		PublicPool: "default",
	}

	clientset := fake.NewSimpleClientset()
	blendedset := blendedfake.NewSimpleClientset()
	informer := informers.NewSharedInformerFactory(clientset, 0)

	controller := NewController(cfg, clientset, blendedset, informer.Core().V1().Namespaces())
	go informer.Start(ctx.Done())
	assert.Nil(t, controller.Run(ctx, cfg.Threads))

	pool := &blendedv1.Pool{
		ObjectMeta: metav1.ObjectMeta{
			Name: cfg.PrivatePool,
		},
		Spec: blendedv1.PoolSpec{
			Addresses:                 []string{"172.22.132.10-172.22.132.15"},
			IgnoreNamespaces:          []string{},
			IgnoreNamespaceAnnotation: false,
			AssignToNamespace:         true,
		},
	}
	_, err := blendedset.InwinstackV1().Pools().Create(pool)
	assert.Nil(t, err)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}

	ip := &blendedv1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s", uuid.NewUUID()),
			Namespace: ns.Name,
		},
		Spec: blendedv1.IPSpec{
			PoolName: pool.Name,
		},
		Status: blendedv1.IPStatus{
			Phase:          blendedv1.IPActive,
			Address:        "172.22.132.11",
			LastUpdateTime: metav1.NewTime(time.Now()),
		},
	}
	_, err = blendedset.InwinstackV1().IPs(ns.Name).Create(ip)
	assert.Nil(t, err)

	_, err = clientset.CoreV1().Namespaces().Create(ns)
	assert.Nil(t, err)

	// Test for creating
	failed := true
	for start := time.Now(); time.Since(start) < timeout; {
		gns, err := clientset.CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{})
		assert.Nil(t, err)
		if address, ok := gns.Annotations[constants.LatestIPKey]; ok {
			assert.Equal(t, ip.Status.Address, address)
			failed = false
			break
		}
	}
	assert.Equal(t, false, failed, "cannot get the private IP.")

	// Test for the number is 0
	gns, err := clientset.CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{})
	assert.Nil(t, err)

	gns.Annotations[constants.NumberOfIPKey] = strconv.Itoa(0)
	_, err = clientset.CoreV1().Namespaces().Update(gns)
	assert.Nil(t, err)

	failed = true
	for start := time.Now(); time.Since(start) < timeout; {
		gns, err := clientset.CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{})
		assert.Nil(t, err)
		if _, ok := gns.Annotations[constants.LatestIPKey]; !ok {
			assert.Equal(t, "0", gns.Annotations[constants.NumberOfIPKey])
			assert.Equal(t, "", gns.Annotations[constants.IPsKey])
			failed = false
			break
		}
	}
	assert.Equal(t, false, failed, "failed to delete the IP.")

	cancel()
	controller.Stop()
}
