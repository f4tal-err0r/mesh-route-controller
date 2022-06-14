# sre-mesh-route-generator

### What is this?

This is a basic kubernetes controller that is built to handle a long running issue in current versions of istio (<= 1.14) that disallows more than one Kubernetes virtualservice from managing internal mesh gateway routes (https://github.com/istio/istio/issues/22997). This issue will be resolved by HTTPRoute CRDs which we will transition to after it exits beta and the managed Anthos Service Mesh upgrades us to the version that this will be availaible.

### How does it work?

The controllers main logic actions on three specific events. Creation, Updates and Deletions of any virtualservice manifest in the cluster with the label `bc-network: edge`. This denotes paths that will be servicing inbound traffic. The routes are then merged and managed within the `infra/mesh-routing` virtualservice. 

### Compatibility

- `istio.io/v1beta1` api

- Go 1.18+


### Test/Build

You can run a `skaffold build` or `skaffold dev` for debugging in lower environments.

`go test -v` will run the test suite. 