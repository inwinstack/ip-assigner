package operator

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
	inwinclientset "github.com/inwinstack/blended/client/clientset/versioned/typed/inwinstack/v1"
	opkit "github.com/inwinstack/operator-kit"
	"github.com/kairen/ip-assigner/pkg/k8sutil"
	"github.com/kairen/ip-assigner/pkg/operator/service"
	"k8s.io/api/core/v1"
	apiextensionsclients "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
)

type Flag struct {
	Kubeconfig       string
	IgnoreNamespaces []string
}

type Operator struct {
	ctx        *opkit.Context
	controller *service.Controller
	flag       *Flag
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
	ctx, clientset, err := o.initContextAndClient()
	if err != nil {
		return err
	}
	o.controller = service.NewController(ctx, clientset, o.flag.IgnoreNamespaces)
	o.ctx = ctx
	return nil
}

func (o *Operator) initContextAndClient() (*opkit.Context, inwinclientset.InwinstackV1Interface, error) {
	glog.V(2).Info("Initialize the operator context and client.")

	config, err := k8sutil.GetRestConfig(o.flag.Kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get Kubernetes config. %+v", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get Kubernetes client. %+v", err)
	}

	extensionsclient, err := apiextensionsclients.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create Kubernetes API extension clientset. %+v", err)
	}

	inwinclient, err := inwinclientset.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create blended clientset. %+v", err)
	}

	ctx := &opkit.Context{
		Clientset:             client,
		APIExtensionClientset: extensionsclient,
		Interval:              interval,
		Timeout:               timeout,
	}
	return ctx, inwinclient, nil
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
