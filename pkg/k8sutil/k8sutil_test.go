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
package k8sutil

import (
	"testing"

	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	fake "github.com/inwinstack/blended/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newIP(name, poolName string) inwinv1.IP {
	return inwinv1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: inwinv1.IPSpec{
			PoolName:        poolName,
			UpdateNamespace: true,
		},
	}
}

func TestFilterIPsByPool(t *testing.T) {
	expected := &inwinv1.IPList{
		Items: []inwinv1.IP{
			newIP("test1", "default"),
			newIP("test4", "default"),
		},
	}

	ips := &inwinv1.IPList{
		Items: []inwinv1.IP{
			newIP("test1", "default"),
			newIP("test2", "test"),
			newIP("test3", "internet"),
			newIP("test4", "default"),
		},
	}

	pool := NewDefaultPool(
		"172.22.132.150-172.22.132.200",
		[]string{"default", "kube-system", "kube-public"},
		false, true,
	)
	FilterIPsByPool(ips, pool)
	assert.Equal(t, expected, ips)
}

func TestNewIPWithNamespace(t *testing.T) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}

	client := fake.NewSimpleClientset()

	ip := NewIPWithNamespace(ns, "default")
	createIP, err := client.InwinstackV1().IPs(ns.Name).Create(ip)
	assert.Nil(t, err)
	assert.NotNil(t, createIP)

	referNS := createIP.OwnerReferences[0]
	assert.Equal(t, ns.Name, referNS.Name)
}
