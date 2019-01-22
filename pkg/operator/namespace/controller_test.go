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
	corefake "k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/uuid"
)

func TestController(t *testing.T) {
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
		},
	}
	createNS, nserr := coreClient.CoreV1().Namespaces().Create(ns)
	assert.Nil(t, nserr)

	ip := &inwinv1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s", uuid.NewUUID()),
			Namespace: ns.Name,
		},
		Spec: inwinv1.IPSpec{
			PoolName:             pool.Name,
			MarkNamespaceRefresh: false,
		},
		Status: inwinv1.IPStatus{
			Phase:          inwinv1.IPActive,
			Address:        "172.22.132.150",
			LastUpdateTime: metav1.NewTime(time.Now()),
		},
	}
	_, iperr := client.InwinstackV1().IPs(ns.Name).Create(ip)
	assert.Nil(t, iperr)

	ctx := &opkit.Context{
		Clientset:             coreClient,
		APIExtensionClientset: extensionsClient,
		Interval:              500 * time.Millisecond,
		Timeout:               60 * time.Second,
	}
	controller := NewController(ctx, client, 5)

	// Test onAdd
	controller.onAdd(createNS)

	onAddNS, err := coreClient.CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	onAddNS.Annotations[constants.AnnKeyNamespaceRefresh] = "true"

	// Test onUpdate
	controller.onUpdate(createNS, onAddNS)

	onUpdateNS, err := coreClient.CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, ip.Status.Address, onUpdateNS.Annotations[constants.AnnKeyLatestIP])
	onUpdateNS.Annotations[constants.AnnKeyNumberOfIP] = "0"
	onUpdateNS.Annotations[constants.AnnKeyNamespaceRefresh] = "true"

	// Test delete IP
	controller.onUpdate(onAddNS, onUpdateNS)

	_, iperr2 := client.InwinstackV1().IPs(ns.Name).Get(ip.Name, metav1.GetOptions{})
	assert.NotNil(t, iperr2)

	deleteNS, err := coreClient.CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, "", deleteNS.Annotations[constants.AnnKeyLatestIP])
}
