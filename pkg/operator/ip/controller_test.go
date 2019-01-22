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
	"testing"
	"time"

	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	fake "github.com/inwinstack/blended/client/clientset/versioned/fake"
	"github.com/inwinstack/ip-assigner/pkg/constants"
	opkit "github.com/inwinstack/operator-kit"
	"k8s.io/api/core/v1"
	extensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	corefake "k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
)

func TestIPController(t *testing.T) {
	client := fake.NewSimpleClientset()
	coreClient := corefake.NewSimpleClientset()
	extensionsClient := extensionsfake.NewSimpleClientset()

	pool := &inwinv1.Pool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: inwinv1.PoolSpec{
			Addresses:                 []string{"172.22.132.150-172.22.132.200"},
			IgnoreNamespaces:          []string{"default", "kube-system", "kube-public"},
			IgnoreNamespaceAnnotation: false,
			AssignToNamespace:         true,
		},
	}
	_, poolerr := client.InwinstackV1().Pools().Create(pool)
	assert.Nil(t, poolerr)

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Annotations: map[string]string{
				constants.AnnKeyPoolName:   pool.Name,
				constants.AnnKeyNumberOfIP: "1",
			},
		},
		Status: v1.NamespaceStatus{
			Phase: v1.NamespaceActive,
		},
	}
	_, err := coreClient.CoreV1().Namespaces().Create(ns)
	assert.Nil(t, err)

	ip := &inwinv1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s", uuid.NewUUID()),
			Namespace: ns.Name,
		},
		Spec: inwinv1.IPSpec{
			PoolName: pool.Name,
		},
		Status: inwinv1.IPStatus{
			Phase:          inwinv1.IPActive,
			Address:        "172.22.132.150",
			LastUpdateTime: metav1.NewTime(time.Now()),
		},
	}
	createIP, iperr := client.InwinstackV1().IPs(ns.Name).Create(ip)
	assert.Nil(t, iperr)

	ctx := &opkit.Context{
		Clientset:             coreClient,
		APIExtensionClientset: extensionsClient,
		Interval:              500 * time.Millisecond,
		Timeout:               60 * time.Second,
	}
	controller := NewController(ctx, client)

	// Test onAdd
	controller.onAdd(createIP)

	onAddNs, err := coreClient.CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, "172.22.132.150", onAddNs.Annotations[constants.AnnKeyLatestIP])
	assert.Equal(t, "172.22.132.150", onAddNs.Annotations[constants.AnnKeyIPs])

	// Test onUpate
	createIP.Annotations = map[string]string{constants.AnnKeyDirtyResource: "true"}
	updateIP, err := client.InwinstackV1().IPs(ns.Name).Update(createIP)
	assert.Nil(t, err)

	controller.onUpdate(createIP, updateIP)

	onUpdateIPs, err := client.InwinstackV1().IPs(ns.Name).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(onUpdateIPs.Items))
	// Test onDelete
	controller.onDelete(updateIP)

	onDeleteNs, err := coreClient.CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, "", onDeleteNs.Annotations[constants.AnnKeyLatestIP])
	assert.Equal(t, "", onDeleteNs.Annotations[constants.AnnKeyIPs])
}
