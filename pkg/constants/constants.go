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

package constants

const Finalizer = "kubernetes"

const PolicyPrefix = "k8s"

const (
	// DefaultNumberOfIP represents the number of IP for a Namespace.
	DefaultNumberOfIP = 1
	// IPsKey is the key of annotation for displaying allocated IPs.
	IPsKey = "inwinstack.com/allocated-ips"
	// LatestIPKey is the key of annotation for displaying the latest of allocated IP.
	LatestIPKey = "inwinstack.com/allocated-latest-ip"
	// NumberOfIPKey is the key of annotation for representing the number of IP needs to allocate.
	NumberOfIPKey = "inwinstack.com/allocate-ip-number"
	// PrivatePoolKey is the key of annotation for the private pool for assigning IP.
	PrivatePoolKey = "inwinstack.com/allocate-pool-name"
	// PublicPoolKey is the key of annotation for the public pool for assigning IP.
	PublicPoolKey = "inwinstack.com/external-pool"
	// PublicIPKey is the key of annotation for displaying allocated public IP.
	PublicIPKey = "inwinstack.com/allocated-public-ip"
	// LatestPoolKey is the key of annotation for displaying the latest pool name.
	LatestPoolKey = "inwinstack.com/latest-pool"
)
