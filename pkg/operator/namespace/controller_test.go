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
	"testing"
	"time"

	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	fake "github.com/inwinstack/blended/client/clientset/versioned/fake"
	"github.com/inwinstack/ip-assigner/pkg/config"
	"github.com/inwinstack/ip-assigner/pkg/constants"
	opkit "github.com/inwinstack/operator-kit"
	"k8s.io/api/core/v1"
	extensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corefake "k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
)

func TestNamespaceController(t *testing.T) {
	client := fake.NewSimpleClientset()
	coreClient := corefake.NewSimpleClientset()
	extensionsClient := extensionsfake.NewSimpleClientset()

	conf := &config.OperatorConfig{
		PoolName:         "default",
		Addresses:        []string{"172.22.132.150-172.22.132.200"},
		IgnoreNamespaces: []string{"kube-system", "default", "kube-public"},
		KeepUpdate:       true,
	}

	pool := &inwinv1.Pool{
		ObjectMeta: metav1.ObjectMeta{
			Name: conf.PoolName,
		},
		Spec: inwinv1.PoolSpec{
			Addresses:                 conf.Addresses,
			IgnoreNamespaces:          conf.IgnoreNamespaces,
			IgnoreNamespaceAnnotation: false,
			AssignToNamespace:         true,
		},
	}
	_, poolerr := client.InwinstackV1().Pools().Create(pool)
	assert.Nil(t, poolerr)

	testNs := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
	createTestNs, nserr := coreClient.CoreV1().Namespaces().Create(testNs)
	assert.Nil(t, nserr)

	defaultNs := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
	}
	createDefaultNs, nserr := coreClient.CoreV1().Namespaces().Create(defaultNs)
	assert.Nil(t, nserr)

	ctx := &opkit.Context{
		Clientset:             coreClient,
		APIExtensionClientset: extensionsClient,
		Interval:              500 * time.Millisecond,
		Timeout:               60 * time.Second,
	}
	controller := NewController(ctx, client, conf)

	// Test onAdd for default namespace
	controller.onAdd(createDefaultNs)

	onAddDefaultNS, err := coreClient.CoreV1().Namespaces().Get(defaultNs.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Nil(t, onAddDefaultNS.Annotations)

	// Test onAdd for test namespace
	controller.onAdd(createTestNs)

	onAddTestNS, err := coreClient.CoreV1().Namespaces().Get(testNs.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, onAddTestNS.Annotations)
	assert.Equal(t, pool.Name, onAddTestNS.Annotations[constants.AnnKeyPoolName])
	assert.Equal(t, "1", onAddTestNS.Annotations[constants.AnnKeyNumberOfIP])

	onAddIPs, err := client.InwinstackV1().IPs(testNs.Name).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(onAddIPs.Items))

	// Test onUpdate
	onAddTestNS.Annotations[constants.AnnKeyNumberOfIP] = "0"
	controller.onUpdate(createTestNs, onAddTestNS)

	onUpdateIPs, err := client.InwinstackV1().IPs(testNs.Name).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(onUpdateIPs.Items))
	assert.Equal(t, "true", onUpdateIPs.Items[0].Annotations[constants.AnnKeyDirtyResource])
}
