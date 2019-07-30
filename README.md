[![Build Status](https://travis-ci.org/inwinstack/ip-assigner.svg?branch=master)](https://travis-ci.org/inwinstack/ip-assigner) [![codecov](https://codecov.io/gh/inwinstack/ip-assigner/branch/master/graph/badge.svg)](https://codecov.io/gh/inwinstack/ip-assigner) [![Docker Pulls](https://img.shields.io/docker/pulls/inwinstack/ip-assigner.svg)](https://hub.docker.com/r/inwinstack/ip-assigner/) ![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)

# IP Assigner
A controller that automatically assigns the IP address to Kubernetes Namespace(Private) and Services(Public).

### Requirements
IP Assigner depend on IPAM, you can see more details from [IPAM GitHub](https://github.com/inwinstack/ipam).

#### Compatibility Matrix

| IPAM version | 0.7.x | 0.6.x | 0.5.x | 
|--------------|-------|-------|-------|
| 0.4.x        | Y     | Y     | X     |
| 0.3.x        | Y     | Y     | X     |
| 0.2.x        | Y     | Y     | Y     |

## Building from Source
Clone repo into your go path under `$GOPATH/src`:
```sh
$ git clone https://github.com/inwinstack/ip-assigner $GOPATH/src/github.com/inwinstack/ip-assigner
$ cd $GOPATH/src/github.com/inwinstack/ip-assigner
$ make dep
$ make
```

## Debug out of the cluster
Run the following command to debug:
```sh
$ go run cmd/main.go \
    -v=2 \
    --logtostderr \
    --kubeconfig $HOME/.kube/config \
    --pool-configmap-names=default,internet
```

## Deploy in the cluster
Run the following command to deploy operator:
```sh
$ kubectl apply -f deploy/
$ kubectl -n kube-system get po -l app=ip-assigner
```
