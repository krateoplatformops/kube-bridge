# Kube Bridge

Krateo Control Plane Kubernetes Bridge.

---

## How to try locally for development

Run a local Kubernetes cluster using [kind](https://github.com/kubernetes-sigs/kind):

```sh
$ make kind.up
```

With [kind](https://github.com/kubernetes-sigs/kind) up and running, install [krateo runtime](https://github.com/krateoplatformops/krateo):

```sh
$ krateo init
```

Apply [RBAC permissions](./manifests/rbac.yaml) for this service:

```sh
$ kubectl apply -f manifests/rbac.yaml -n krateo-system
```

[Deploy the service](./manifests/service.yaml):

```sh
$ kubectl apply -f manifests/service.yaml -n krateo-system
```

To be able to invoke this service API using [`curl`](https://github.com/curl/curl) from your machine, open another terminal and type:

```sh
$ kubectl port-forward -n krateo-system service/kube-bridge 8171:8171
```

Try the _apply_ endpoint using a sample payload:

```sh
$ curl --data @testdata/sample.json -H \
   "content-type:application/json" http://localhost:8171/apply
```



