# IP Assigner
An operator to assign the external IP for Kubernetes Service.

## Building from Source
Clone repo into your go path under `$GOPATH/src`:
```sh
$ git clone https://github.com/kairen/ip-assigner $GOPATH/src/github.com/kairen/ip-assigner
$ cd $GOPATH/src/github.com/kairen/ip-assigner
$ make dep
$ make
```

## Debug out of the cluster
Run the following command to debug:
```sh
$ go run cmd/main.go \
    --kubeconfig $HOME/.kube/config \
    --logtostderr \
    --ignore-namespaces=kube-system,default,kube-public \
    -v=2
```

## Deploy in the cluster
Run the following command to deploy operator:
```sh
$ kubectl apply -f deploy/
$ kubectl -n kube-system get po -l app=ip-assigner
```