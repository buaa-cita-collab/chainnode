/*
Copyright 2021 buaa-cita.

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

package controllers

import (
	"context"
	"fmt"

	citacloudv1 "github.com/buaa-cita/chainnode/api/v1"
	appv1 "k8s.io/api/apps/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	corev1 "k8s.io/api/core/v1"
)

// ChainNodeReconciler reconciles a ChainNode object
type ChainNodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=citacloud.buaa.edu.cn,resources=chainnodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=citacloud.buaa.edu.cn,resources=chainnodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=citacloud.buaa.edu.cn,resources=chainnodes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ChainNode object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile

// 节点级别的配置，每个节点在符合当前链的治理下，选择自己的个性化配置
/*
	pvcName // 此节点配置文件的路径
	name // 节点的名字
	chainconfig // 指定使用的链config
	kms_password  // 节点的kms服务的密码
	network-key // 节点的私钥
	六个微服务的配置：包括：
		is_std_out // 是否将日志输出到标准输出
		log-level // 日志等级
		以及其他可以各个节点不同的配置

	service.port //服务的端口
	service.eipName

	// loadbanlancer，可以交由运维的人员来处理，
		service.endIP //多集群
		service.interface
		service.startIP

*/

func (r *ChainNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("start reconcile")
	var chainNode citacloudv1.ChainNode
	logger.Info("Reconcile called really do")

	// fetch chainNode
	if err := r.Get(ctx, req.NamespacedName, &chainNode); err != nil {
		logger.Error(err, "get ChainNode failed")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// fetch chainConfig
	var configName string
	if chainNode.Spec.ConfigName == "" {
		// not configName not setted
		configName = "chainconfig-sample"
	} else {
		configName = chainNode.Spec.ConfigName
	}
	
	configKey := types.NamespacedName{
		Namespace: req.NamespacedName.Namespace,
		Name:      configName}
	var chainConfig citacloudv1.ChainConfig
	if err := r.Get(ctx, configKey, &chainConfig); err != nil {
		if apierror.IsNotFound(err) {
			//dont show too much info if it is a not found error
			logger.Info("can not find chainconfig")
		} else {
			logger.Error(err, "get chainconfig failed")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// test if the Deployment exists
	deploymentName := req.Name + "_deployment"
	var deployment appv1.Deployment
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      deploymentName,
	}, &deployment); err != nil {
		// can not find deployment
		if apierror.IsNotFound(err) {
			//build deployment
			if errBuild := buildNodeDeployment(chainNode, chainConfig, &deployment); errBuild != nil {
				logger.Error(errBuild, "Failed building node Deployment")
				// return nil to avoid rerun reconcile
				return ctrl.Result{}, nil
			}
			if errCreate := r.Create(ctx, &deployment); errCreate != nil {
				logger.Error(err, "Failed create Deployment")
				// return nil to avoid rerun reconcile
				return ctrl.Result{}, nil
			}
		} else {
			// other error
			logger.Error(err, "can not find deployment")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	//deployment存在，应该对其进行更新
	var newDeploy appv1.Deployment
	if errBuild := buildNodeDeployment(chainNode, chainConfig, &newDeploy); errBuild != nil {
		logger.Error(errBuild, "Failed building node Deployment")
		// return nil to avoid rerun reconcile
		return ctrl.Result{}, nil
	}

	deployment.Spec = newDeploy.Spec

	if err := r.Update(ctx, &deployment); err != nil {
		logger.Error(err, "Update ChainNode failed")
		return ctrl.Result{}, err
	}


	// test if the kmsSecret exists
	kmsSecretName := req.Name + "_kms_secret"
	var kmsSecret corev1.Secret

	if err := r.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      kmsSecretName,
	}, &secret); err != nil {
		// can not find Secret
		if apierror.IsNotFound(err) {
			//build Secret
			if errBuild := buildNodeKmsSecret(chainNode, chainConfig, &kmssecret); errBuild != nil {
				logger.Error(errBuild, "Failed building node Secret")
				// return nil to avoid rerun reconcile
				return ctrl.Result{}, nil
			}
			if errCreate := r.Create(ctx, &kmssecret); errCreate != nil {
				logger.Error(err, "Failed create Secret")
				// return nil to avoid rerun reconcile
				return ctrl.Result{}, nil
			}
		} else {
			// other error
			logger.Error(err, "can not find Secret")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	//kmsSecret存在，应该对其进行更新
	var newKmsSecret corev1.Secret
	if errBuild := buildNodeKmsSecret(chainNode, chainConfig, &newKmsSecret); errBuild != nil {
		logger.Error(errBuild, "Failed building node Deployment")
		// return nil to avoid rerun reconcile
		return ctrl.Result{}, nil
	}

	kmsSecret.Spec = newKmsSecret.Spec

	if err := r.Update(ctx, &kmsSecret); err != nil {
		logger.Error(err, "Update ChainNode failed")
		return ctrl.Result{}, err
	}


	// test if the networkSecret exists
	networkSecretName := req.Name + "_network_secret"
	var networkSecret corev1.Secret

	if err := r.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      networkSecretName,
	}, &secret); err != nil {
		// can not find Secret
		if apierror.IsNotFound(err) {
			//build Secret
			if errBuild := buildNodenNtworkSecret(chainNode, chainConfig, &networksecret); errBuild != nil {
				logger.Error(errBuild, "Failed building node Secret")
				// return nil to avoid rerun reconcile
				return ctrl.Result{}, nil
			}
			if errCreate := r.Create(ctx, &networksecret); errCreate != nil {
				logger.Error(err, "Failed create Secret")
				// return nil to avoid rerun reconcile
				return ctrl.Result{}, nil
			}
		} else {
			// other error
			logger.Error(err, "can not find Secret")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	//networkSecret存在，应该对其进行更新
	var newNetworkSecret corev1.Secret
	if errBuild := buildNodeNetworkSecret(chainNode, chainConfig, &newNetworkSecret); errBuild != nil {
		logger.Error(errBuild, "Failed building network Secret")
		// return nil to avoid rerun reconcile
		return ctrl.Result{}, nil
	}

	networkSecret.Spec = newNetworkSecret.Spec

	if err := r.Update(ctx, &networkSecret); err != nil {
		logger.Error(err, "Update ChainNode failed")
		return ctrl.Result{}, err
	}


	fmt.Println("old status", "chainNode", chainNode.Status.ChainName)
	chainNode.Status.ChainName = chainConfig.Spec.ChainName

	// if err := r.Update(ctx, &chainNode); err != nil {
	// 	logger.Error(err, "update chainNode failed")
	// }

	if err := r.Status().Update(ctx, &chainNode); err != nil {
		logger.Error(err, "update chainNode failed")
	}

	fmt.Println("new status", "chainNode", chainNode.Status.ChainName)

	return ctrl.Result{}, nil
}

func buildNodeDeployment(chainNode citacloudv1.ChainNode,
	chainConfig citacloudv1.ChainConfig,
	pdeployment *appv1.Deployment) error {

	// init parameters
	chainName := chainConfig.Spec.ChainName
	nodeID := chainNode.Spec.NodeID
	nodeName := chainName + "_" + nodeID
	replicas := int32(1)
	pTrue := new(bool) // a pointer point to true
	*pTrue = true

	networkImage := chainConfig.Spec.NetworkImage
	consensusImage := chainConfig.Spec.ConsensusImage
	executorImage := chainConfig.Spec.ExecutorImage
	storageImage := chainConfig.Spec.StorageImage
	controllerImage := chainConfig.Spec.ControllerImage
	kmsImage := chainConfig.Spec.KmsImage

	// build deployment
	deployment := appv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "app/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      chainName + "_" + nodeID,
			Namespace: "default",
			Labels: map[string]string{
				"node_name":  nodeName,
				"chain_name": chainName,
			},
		},
		Spec: appv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"nodes_name": nodeName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"node_name":  nodeName,
						"chain_name": chainName,
					},
				},
				Spec: corev1.PodSpec{
					ShareProcessNamespace: pTrue,
					Containers: []corev1.Container{
						{
							Image:           "syncthing/syncthing:latest",
							ImagePullPolicy: "Always",
							Name:            "syncthing",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 22000,
									Protocol:      "TCP",
									Name:          "sync",
								},
								{
									ContainerPort: 8384,
									Protocol:      "TCP",
									Name:          "gui",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									SubPath:   "cita-cloud/test-chain/node" + nodeID,
									MountPath: "/var/syncthing",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PUID",
									Value: "0",
								},
								{
									Name:  "PUID",
									Value: "0",
								},
							},
						},
						{
							Image:           networkImage,//"citacloud/network_direct",
							ImagePullPolicy: "Always",
							Name:            "network",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 40000,
									Protocol:      "TCP",
									Name:          "network",
								},
								{
									ContainerPort: 50000,
									Protocol:      "TCP",
									Name:          "grpc",
								},
							},
							Command: []string{
								"sh",
								"-c",
								"network run -p 50000",
							},
							WorkingDir: "/data",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									SubPath:   "cita-cloud/test-chain/node" + nodeID,
									MountPath: "/data",
								},
								{
									Name:      "network-key",
									MountPath: "/network",
									ReadOnly:  true,
								},
							},
						},
						{
							Image:           consensusImage,//"citacloud/consensus_bft",
							ImagePullPolicy: "Always",
							Name:            "consensus",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 50001,
									Protocol:      "TCP",
									Name:          "grpc",
								},
							},
							Command: []string{
								"sh",
								"-c",
								"consensus run -p 50001",
							},
							WorkingDir: "/data",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									SubPath:   "cita-cloud/test-chain/node" + nodeID,
									MountPath: "/data",
								},
							},
						},
						{
							Image:           executorImage,//"citacloud/executor_evm",
							ImagePullPolicy: "Always",
							Name:            "executor",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 50002,
									Protocol:      "TCP",
									Name:          "grpc",
								},
							},
							Command: []string{
								"sh",
								"-c",
								"executor run -p 50002",
							},
							WorkingDir: "/data",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									SubPath:   "cita-cloud/test-chain/node" + nodeID,
									MountPath: "/data",
								},
							},
						},
						{
							Image:           storageImage,//"citacloud/storage_rocksdb",
							ImagePullPolicy: "Always",
							Name:            "storage",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 50003,
									Protocol:      "TCP",
									Name:          "grpc",
								},
							},
							Command: []string{
								"sh",
								"-c",
								"stroage run -p 50003",
							},
							WorkingDir: "/data",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									SubPath:   "cita-cloud/test-chain/node" + nodeID,
									MountPath: "/data",
								},
							},
						},
						{
							Image:           controllerImage,//"citacloud/controller",
							ImagePullPolicy: "Always",
							Name:            "controller",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 50004,
									Protocol:      "TCP",
									Name:          "grpc",
								},
							},
							Command: []string{
								"sh",
								"-c",
								"controller run -p 50004",
							},
							WorkingDir: "/data",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									SubPath:   "cita-cloud/test-chain/node" + nodeID,
									MountPath: "/data",
								},
							},
						},
						{
							Image:           kmsImage,//"citacloud/kms_sm",
							ImagePullPolicy: "Always",
							Name:            "kms",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 50005,
									Protocol:      "TCP",
									Name:          "grpc",
								},
							},
							Command: []string{
								"sh",
								"-c",
								"kms run -p 50004 -k /kms/key_file",
							},
							WorkingDir: "/data",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									SubPath:   "cita-cloud/test-chain/node" + nodeID,
									MountPath: "/data",
								},
								{
									Name:      "kms-key",
									MountPath: "/kms",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "kms-key",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "kms-secret-" + chainName,
								},
							},
						},
						{
							Name: "network-key",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: chainName + "-" + nodeID + "network-secret",
								},
							},
						},
						{
							Name: "datadir",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "local-pvc",
								},
							},
						},
					},
				},
			},
		},
	}
	deployment.DeepCopyInto(pdeployment)
	return nil
}

func buildNodeKmsSecret(chainNode citacloudv1.ChainNode, chainConfig citacloudv1.ChainConfig, psecret *corev1.Secret) error {
	
	// init parameters
	chainName := chainConfig.Spec.ChainName
	nodeID := chainNode.Spec.NodeID
	nodeName := chainName + "_" + nodeID
	replicas := int32(1)

	// build secret
	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kms-secret-" + chainName,
			Labels: map[string]string{
				"chain_name":  chainName,
				"secret_level": "chainconfig",
			},
		},
		Type: corev1.SecretType{
			Type: "Opaque",
		},
		Date: map[string]string{
			"key_file":  "MTIzNDU2",
		}
	}
	secret.DeepCopyInto(psecret)
	return nil
}

func buildNodeNetworkSecret(chainNode citacloudv1.ChainNode, chainConfig citacloudv1.ChainConfig, psecret *corev1.Secret) error {
	
	// init parameters
	chainName := chainConfig.Spec.ChainName
	nodeID := chainNode.Spec.NodeID
	nodeName := chainName + "_" + nodeID

	networkKey := chainnode.Spec.NetworkKey

	// build secret
	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      chainName + "-" + nodeID + "-network-secret",
			Labels: map[string]string{
				"node_id":  nodeID,
			},
		},
		Type: corev1.SecretType{
			Type: "Opaque",
		},
		Date: map[string]string{
			"network-key":  networkKey,//"MHg2N2ZiMDhlM2NkMThhMGQ2NDVlNGVhZmY0MTY2YmVhMTc4MzIyODJlMjNkMzQxNGJmNzUwYjhjZjQ3YjkzZDQz",
		}
	}
	secret.DeepCopyInto(psecret)
	return nil
}



func mapFunc(obj client.Object) []ctrl.Request {
	fmt.Println("map func called", obj.GetNamespace(), obj.GetName())
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{Namespace: obj.GetNamespace(), Name: "chainnode-sample"}},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChainNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&citacloudv1.ChainNode{}).
		Watches(&source.Kind{Type: &citacloudv1.ChainConfig{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []ctrl.Request {
				// fmt.Println("map func called", obj.GetNamespace(), obj.GetName())
				ctx := context.Background()
				var nodeList citacloudv1.ChainNodeList
				reqList := make([]reconcile.Request, 0)
				if err := r.List(ctx, &nodeList); err != nil {
					fmt.Println(err)
					return reqList
				}
				for _, node := range nodeList.Items {

					reqList = append(reqList, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: node.GetNamespace(),
							Name:      node.GetName(),
						}})
					//fmt.Println("req", node.GetName())
				}
				return reqList
			})).
		//Watches(&source.Kind{Type: &citacloudv1.ChainConfig{}}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
