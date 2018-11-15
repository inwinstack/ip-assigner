package main

import (
	goflag "flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/kairen/ip-assigner/pkg/operator"
	"github.com/kairen/ip-assigner/pkg/version"
	flag "github.com/spf13/pflag"
)

var (
	kubeconfig string
	namespaces []string
	ver        bool
)

func parserFlags() {
	flag.StringVarP(&kubeconfig, "kubeconfig", "", "", "Absolute path to the kubeconfig file.")
	flag.StringSliceVarP(&namespaces, "ignore-namespaces", "", nil, "Which namespaces will be ignored.")
	flag.BoolVarP(&ver, "version", "", false, "Display the version")
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()
}

func main() {
	defer glog.Flush()
	parserFlags()

	if ver {
		fmt.Fprintf(os.Stdout, "%s\n", version.GetVersion())
		os.Exit(0)
	}

	glog.Infof("Starting IP assigner...")

	f := &operator.Flag{
		Kubeconfig:       kubeconfig,
		IgnoreNamespaces: namespaces,
	}
	op := operator.NewMainOperator(f)
	if err := op.Initialize(); err != nil {
		glog.Fatalf("Error initing operator instance: %+v.\n", err)
	}

	if err := op.Run(); err != nil {
		glog.Fatalf("Error serving operator instance: %+v.\n", err)
	}
}
