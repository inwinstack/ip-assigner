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

package operator

import (
	"context"
	"fmt"
	"time"

	blended "github.com/inwinstack/blended/generated/clientset/versioned"
	"github.com/inwinstack/ip-assigner/pkg/config"
	"github.com/inwinstack/ip-assigner/pkg/operator/namespace"
	"github.com/inwinstack/ip-assigner/pkg/operator/service"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

const defaultSyncTime = time.Second * 30

// Operator represents an operator context
type Operator struct {
	clientset  kubernetes.Interface
	blendedset blended.Interface
	informer   informers.SharedInformerFactory

	cfg       *config.Config
	namespace *namespace.Controller
	service   *service.Controller
}

// New creates an instance of the operator
func New(cfg *config.Config, clientset kubernetes.Interface, blendedset blended.Interface) *Operator {
	o := &Operator{cfg: cfg, clientset: clientset, blendedset: blendedset}
	t := defaultSyncTime
	if cfg.SyncSec > 30 {
		t = time.Second * time.Duration(cfg.SyncSec)
	}
	o.informer = informers.NewSharedInformerFactory(clientset, t)
	o.service = service.NewController(cfg, clientset, blendedset, o.informer.Core().V1().Services())
	o.namespace = namespace.NewController(cfg, clientset, blendedset, o.informer.Core().V1().Namespaces())
	return o
}

// Run serves an isntance of the operator
func (o *Operator) Run(ctx context.Context) error {
	go o.informer.Start(ctx.Done())

	if err := o.service.Run(ctx, o.cfg.Threads); err != nil {
		return fmt.Errorf("failed to run Service controller: %s", err.Error())
	}

	if err := o.namespace.Run(ctx, o.cfg.Threads); err != nil {
		return fmt.Errorf("failed to run Namespace controller: %s", err.Error())
	}
	return nil
}

// Stop stops all controllers
func (o *Operator) Stop() {
	o.service.Stop()
	o.namespace.Stop()
}
