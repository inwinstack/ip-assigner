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
	"testing"

	inwinv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	fake "github.com/inwinstack/blended/client/clientset/versioned/fake"
	extensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corefake "k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
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
}
