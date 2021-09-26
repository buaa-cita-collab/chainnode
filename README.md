# Chain-node
CRD and operator designed to achieve CITA-Cloud cloud native

## Background
- Cita-Cloud can develop solutions based on the alliance chain for enterprise scenarios, and directly deliver the chain as a service object to the end, avoiding various problems encountered by current enterprises using the alliance chain.
- The "Alliance chain + cloud native" solution can reuse the cloud native software stack, which will be closer to enterprise applications, and is superior to traditional alliance chain solutions in terms of ease of use. If the enterprise already uses Kubernetes, it can greatly reduce the switching cost of the enterprise.

## Environment
- kubebuilder : v3.1.0
- kubernetes api : v1.21.2
- go : v1.16.6

## Manual
Use kubebuilder scaffolding to write operator for the secondary development of k8s. Cooperating with the realized [charts project](https://github.com/buaa-cita/charts), the operation and maintenance of CITA-Cloud can be carried out. This project mainly realizes the combination of CITA-Cloud and k8s, using CRD to combine the configuration of the microservice module with ConfigMap, and stripping out the variable configuration at the same time, facilitating operation and maintenance adjustment , The specific configuration can refer to the [charts project](https://github.com/buaa-cita/charts).

### Installation Environment
Need an overseas agent, or set up GO China agent
```
# Enable Go Modules function
$ go env -w GO111MODULE=on
$ go env -w  GOPROXY=https://goproxy.cn,direct
$ go env | grep GOPROXY
GOPROXY="https://goproxy.cn"
```
After the environment is configured, you can install it
```
$ make install

/root/chainnode/bin/controller-gen "crd:trivialVersions=true,preserveUnknownFields=false" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
/root/chainnode/bin/kustomize build config/crd | kubectl apply -f -
customresourcedefinition.apiextensions.k8s.io/chainconfigs.citacloud.buaa.edu.cn configured
customresourcedefinition.apiextensions.k8s.io/chainnodes.citacloud.buaa.edu.cn configured
```

### Inject configuration into k8s API Server
```
$ make run

/root/chainnode/bin/controller-gen "crd:trivialVersions=true,preserveUnknownFields=false" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
/root/chainnode/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
go fmt ./...
go vet ./...
go run ./main.go
...
2021-09-26T14:11:23.275+0800	INFO	controller-runtime.manager.controller.chainconfig	start chainconfig reconcile	{"reconciler group": "citacloud.buaa.edu.cn", "reconciler kind": "ChainConfig", "name": "chainconfig-sample", "namespace": "default"}
```

At this time, it has entered the **Reconcile()** function. With the charts project, you can perform operations such as CITA-Cloud operation and maintenance.
