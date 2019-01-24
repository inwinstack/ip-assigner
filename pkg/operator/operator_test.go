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

package operator

import (
	"testing"

	fake "github.com/inwinstack/blended/client/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/inwinstack/ip-assigner/pkg/config"
	"github.com/inwinstack/ip-assigner/pkg/constants"
	"github.com/stretchr/testify/assert"
)

func TestOperator(t *testing.T) {
	client := fake.NewSimpleClientset()

	conf := &config.OperatorConfig{
		PoolName:         "default",
		Addresses:        []string{"172.22.132.150-172.22.132.160"},
		IgnoreNamespaces: []string{"kube-system", "default", "kube-public"},
		KeepUpdate:       true,
	}
	op := NewMainOperator(conf)
	assert.Nil(t, op.createAndUdateDefaultPool(client))

	pool, err := client.InwinstackV1().Pools().Get(constants.DefaultPool, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, conf.Addresses, pool.Spec.Addresses)
	assert.Equal(t, conf.IgnoreNamespaces, pool.Spec.IgnoreNamespaces)
	assert.Equal(t, true, pool.Spec.AssignToNamespace)
	assert.Equal(t, false, pool.Spec.IgnoreNamespaceAnnotation)
	assert.Equal(t, true, pool.Spec.AvoidBuggyIPs)
	assert.Equal(t, false, pool.Spec.AvoidGatewayIPs)

	op.conf.Addresses = []string{"172.22.132.150-172.22.132.160", "172.22.132.161-172.22.132.170"}
	assert.Nil(t, op.createAndUdateDefaultPool(client))

	updatePool, err := client.InwinstackV1().Pools().Get(constants.DefaultPool, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, op.conf.Addresses, updatePool.Spec.Addresses)
}
