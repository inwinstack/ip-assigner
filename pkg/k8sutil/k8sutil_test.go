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

package k8sutil

import (
	"testing"

	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	blendedfake "github.com/inwinstack/blended/generated/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPool(t *testing.T) {
	blendedset := blendedfake.NewSimpleClientset()
	meta := metav1.ObjectMeta{
		Name: "test1",
		Annotations: map[string]string{
			"get.pool": "test",
		},
	}

	pool := &blendedv1.Pool{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
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

	gpool, err := GetPool(blendedset, meta, "get.pool")
	assert.Nil(t, err)
	assert.Equal(t, pool.Name, gpool.Name)
	assert.Equal(t, pool.Spec, gpool.Spec)
}

func TestNewIP(t *testing.T) {
	blendedset := blendedfake.NewSimpleClientset()
	_, err := NewIP(blendedset, "test", "default", "default")
	assert.Nil(t, err)
}

func TestFilterIPsByPool(t *testing.T) {
	newIP := func(name, poolName string) blendedv1.IP {
		return blendedv1.IP{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: blendedv1.IPSpec{
				PoolName: poolName,
			},
		}
	}

	expected := &blendedv1.IPList{
		Items: []blendedv1.IP{
			newIP("test1", "default"),
			newIP("test4", "default"),
		},
	}

	ips := &blendedv1.IPList{
		Items: []blendedv1.IP{
			newIP("test1", "default"),
			newIP("test2", "test"),
			newIP("test3", "internet"),
			newIP("test4", "default"),
		},
	}
	FilterIPsByPool(ips, "default")
	assert.Equal(t, expected, ips)
}
