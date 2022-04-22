# Kube Bridge

Krateo Control Plane Kubernetes Bridge

## How to install

Install [Krateo Runtime](https://github.com/krateoplatformops/krateo):

```sh
$ krateo init
```

Apply [RBAC manifest](./manifests/rbac.yaml):

```sh
$ kubectl apply -f manifests/rbac.yaml
```

Apply [Service manifest](./manifests/service.yaml):

```sh
$ kubectl apply -f manifests/rbac.yaml -n krateo-system
```




