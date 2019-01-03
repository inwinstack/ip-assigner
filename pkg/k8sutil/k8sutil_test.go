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
	"github.com/stretchr/testify/assert"
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

func newPool(name string) *inwinv1.Pool {
	return &inwinv1.Pool{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: inwinv1.PoolSpec{
			Address:                   "172.22.132.150-172.22.132.200",
			IgnoreNamespaces:          []string{"default", "kube-system", "kube-public"},
			IgnoreNamespaceAnnotation: false,
			AutoAssignToNamespace:     true,
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

	pool := newPool("default")
	FilterIPsByPool(ips, pool)
	assert.Equal(t, expected, ips)
}
