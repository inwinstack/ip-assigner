module github.com/inwinstack/ip-assigner

go 1.12

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/inwinstack/blended v0.7.0
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.3.0
	github.com/thoas/go-funk v0.4.0
	k8s.io/api v0.0.0-20190726022912-69e1bce1dad5
	k8s.io/apiextensions-apiserver v0.0.0-20190726024412-102230e288fd // indirect
	k8s.io/apimachinery v0.0.0-20190726022757-641a75999153
	k8s.io/client-go v8.0.0+incompatible
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190620084959-7cf5895f2711
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
)
