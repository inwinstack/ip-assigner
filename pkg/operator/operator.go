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
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
	clientset "github.com/inwinstack/blended/client/clientset/versioned"
	"github.com/inwinstack/ip-assigner/pkg/constants"
	"github.com/inwinstack/ip-assigner/pkg/k8sutil"
	"github.com/inwinstack/ip-assigner/pkg/operator/namespace"
	opkit "github.com/inwinstack/operator-kit"
	"k8s.io/api/core/v1"
	apiextensionsclients "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Flag struct {
	Kubeconfig       string
	Addresses        []string
	IgnoreNamespaces []string
	KeepUpdate       bool
}

type Operator struct {
	flag       *Flag
	ctx        *opkit.Context
	controller *namespace.NamespaceController
}

const (
	initRetryDelay = 10 * time.Second
	interval       = 500 * time.Millisecond
	timeout        = 60 * time.Second
)

func NewMainOperator(flag *Flag) *Operator {
	return &Operator{flag: flag}
}

func (o *Operator) Initialize() error {
	glog.V(2).Info("Initialize the operator resources.")

	ctx, blendedClient, err := o.initContextAndClient()
	if err != nil {
		return err
	}

	if err := o.createAndUdateDefaultPool(blendedClient); err != nil {
		glog.Fatalf("Failed to create default pool. %+v", err)
	}

	o.controller = namespace.NewController(ctx, blendedClient)
	o.ctx = ctx
	return nil
}

func (o *Operator) initContextAndClient() (*opkit.Context, clientset.Interface, error) {
	glog.V(2).Info("Initialize the operator context and client.")

	config, err := k8sutil.GetRestConfig(o.flag.Kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get Kubernetes config. %+v", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get Kubernetes client. %+v", err)
	}

	extensionsClient, err := apiextensionsclients.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create Kubernetes API extension clientset. %+v", err)
	}

	blendedClient, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create blended clientset. %+v", err)
	}

	ctx := &opkit.Context{
		Clientset:             client,
		APIExtensionClientset: extensionsClient,
		Interval:              interval,
		Timeout:               timeout,
	}
	return ctx, blendedClient, nil
}

func (o *Operator) createAndUdateDefaultPool(clientset clientset.Interface) error {
	if o.flag.Addresses == nil && o.flag.IgnoreNamespaces == nil {
		return fmt.Errorf("Miss address and namespaces flag")
	}

	pool, err := clientset.InwinstackV1().Pools().Get(constants.DefaultPool, metav1.GetOptions{})
	if err == nil {
		glog.Infof("The default pool already exists.")

		if o.flag.KeepUpdate {
			glog.Infof("The default pool has updated.")
			pool.Spec.IgnoreNamespaces = o.flag.IgnoreNamespaces
			pool.Spec.Addresses = o.flag.Addresses
			pool.Spec.AvoidBuggyIPs = true
			pool.Spec.AssignToNamespace = true
			pool.Spec.IgnoreNamespaceAnnotation = false
			if _, err := clientset.InwinstackV1().Pools().Update(pool); err != nil {
				return err
			}
		}
		return nil
	}

	pool = k8sutil.NewDefaultPool(o.flag.Addresses, o.flag.IgnoreNamespaces)
	if _, err := clientset.InwinstackV1().Pools().Create(pool); err != nil {
		return err
	}
	glog.Infof("The default pool has created.")
	return nil
}

func (o *Operator) Run() error {
	signalChan := make(chan os.Signal, 1)
	stopChan := make(chan struct{})
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// start watching the resources
	o.controller.StartWatch(v1.NamespaceAll, stopChan)

	for {
		select {
		case <-signalChan:
			glog.Infof("Shutdown signal received, exiting...")
			close(stopChan)
			return nil
		}
	}
}
