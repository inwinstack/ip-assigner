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

package main

import (
	goflag "flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/inwinstack/ip-assigner/pkg/config"
	"github.com/inwinstack/ip-assigner/pkg/operator"
	"github.com/inwinstack/ip-assigner/pkg/version"
	flag "github.com/spf13/pflag"
)

var (
	conf = &config.OperatorConfig{}
	ver  bool
)

func parserFlags() {
	flag.StringVarP(&conf.Kubeconfig, "kubeconfig", "", "", "Absolute path to the kubeconfig file.")
	flag.StringVarP(&conf.PoolName, "pool-name", "", "default", "Define the name of the pool.")
	flag.StringSliceVarP(&conf.Addresses, "pool-addresses", "", nil, "Set default IP pool addresses.")
	flag.StringSliceVarP(&conf.IgnoreNamespaces, "pool-ignore-namespaces", "", nil, "Set default IP pool ignore namespaces.")
	flag.BoolVarP(&conf.KeepUpdate, "update", "", true, "Keep update default pool from flags.")
	flag.IntVarP(&conf.Retry, "retry", "", 10, "The number of retry for failed.")
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

	if conf.Addresses == nil || conf.IgnoreNamespaces == nil || conf.PoolName == "" {
		flag.Usage()
		os.Exit(0)
	}

	glog.Infof("Starting IP assigner...")

	op := operator.NewMainOperator(conf)
	if err := op.Initialize(); err != nil {
		glog.Fatalf("Error initing operator instance: %+v.\n", err)
	}

	if err := op.Run(); err != nil {
		glog.Fatalf("Error serving operator instance: %+v.\n", err)
	}
}
