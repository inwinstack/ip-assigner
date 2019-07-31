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
	blendedv1 "github.com/inwinstack/blended/apis/inwinstack/v1"
	blended "github.com/inwinstack/blended/generated/clientset/versioned"
	"github.com/thoas/go-funk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetPool(blendedset blended.Interface, meta metav1.ObjectMeta, key string) (*blendedv1.Pool, error) {
	poolName := meta.Annotations[key]
	pool, err := blendedset.InwinstackV1().Pools().Get(poolName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func NewIP(blendedset blended.Interface, name, namespace, pool string) (*blendedv1.IP, error) {
	ip := &blendedv1.IP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: blendedv1.IPSpec{
			PoolName: pool,
		},
	}
	return blendedset.InwinstackV1().IPs(namespace).Create(ip)
}

func FilterIPsByPool(ips *blendedv1.IPList, poolName string) {
	items := funk.Filter(ips.Items, func(ip blendedv1.IP) bool {
		return poolName == ip.Spec.PoolName
	})
	ips.Items = items.([]blendedv1.IP)
}
