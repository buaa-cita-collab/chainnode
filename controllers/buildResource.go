package controllers

import (
	"context"
	"errors"

	citacloudv1 "github.com/buaa-cita/chainnode/api/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	// "strconv"
)

func (r *ChainNodeReconciler) reconcileNodeDeployment(
	ctx context.Context,
	chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
	prestartFlag *bool,
) error {

	logger := log.FromContext(ctx)
	nodeName := chainNode.ObjectMeta.Name
	deploymentName := nodeName + "-deployment"

	// determine the required operation
	operation := nothingNeeded
	var deployment appv1.Deployment
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: "default",
		Name:      deploymentName,
	}, &deployment); err != nil {
		// can not find deployment
		if apierror.IsNotFound(err) {
			//build deployment
			operation = buildNeeded
		} else {
			logger.Info("error fetch deployment")
			return err
		}
	} else {
		// Deployment found
		if *prestartFlag {
			operation = rebuildNeeded
		}
	}

	if operation == nothingNeeded {
		return nil
	}

	logger.Info("operation: " + operation)

	// Do the operation
	if operation == buildNeeded ||
		operation == rebuildNeeded {
		if operation == rebuildNeeded {
			logger.Info("enter rebuild")
			if errDelete := r.Delete(
				ctx, &deployment,
				client.PropagationPolicy(
					metav1.DeletePropagationForeground,
				),
			); errDelete != nil {
				logger.Error(errDelete,
					"delete deployment failed")
			} else {
				logger.Info("delete deployment succeed")
			}
		}

		// build deployment
		if errBuild := buildNodeDeployment(chainNode,
			chainConfig, &deployment, deploymentName); errBuild != nil {
			logger.Error(errBuild,
				"Failed building node Deployment")
			return nil
		} else {
			logger.Info("build deployment succeed")
		}

		if errSet := ctrl.SetControllerReference(
			chainNode, &deployment, r.Scheme); errSet != nil {
			logger.Error(errSet,
				"set controller reference failed")
			return nil
		}

		if errCreate := r.Create(ctx, &deployment); errCreate != nil {
			logger.Error(errCreate, "Create deployment failed")
			return nil
		} else {
			logger.Info("create deployment succeed")
		}

		// this operation is moved to buildConfig
		// if operation == rebuildNeeded || operation == buildNeeded {
		// 	// update node count
		// 	chainNode.Status.NodeCount = strconv.Itoa(len(chainConfig.Spec.Nodes))
		// 	if errUpdate := r.Status().Update(ctx, chainNode); errUpdate != nil {
		// 		logger.Error(errUpdate, "updatr status failed")
		// 	} else {
		// 		logger.Info("update status succeed")
		// 	}
		// }
	} else if operation == updateNeeded {
		if errUpdate := r.Update(ctx, &deployment); errUpdate != nil {
			logger.Error(errUpdate,
				"Update deployment failed")
			return nil
		} else {
			logger.Info("update deployment succeed")
		}
	}
	return nil
}

func buildNodeDeployment(chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
	pdeployment *appv1.Deployment,
	deploymentName string) error {

	// init parameters
	chainName := chainConfig.ObjectMeta.Name
	nodeName := chainNode.ObjectMeta.Name
	replicas := int32(1)
	pTrue := new(bool) // a pointer point to true
	*pTrue = true

	// Checking configuration
	networkImage := chainConfig.Spec.NetworkImage
	if networkImage == "" {
		return errors.New("Network image should not be void")
	}
	consensusImage := chainConfig.Spec.ConsensusImage
	if consensusImage == "" {
		return errors.New("Consensus image should not be void")
	}
	executorImage := chainConfig.Spec.ExecutorImage
	if executorImage == "" {
		return errors.New("Executor image should not be void")
	}
	storageImage := chainConfig.Spec.StorageImage
	if storageImage == "" {
		return errors.New("Storage image should not be void")
	}
	controllerImage := chainConfig.Spec.ControllerImage
	if controllerImage == "" {
		return errors.New("Controller image should not be void")
	}
	kmsImage := chainConfig.Spec.KmsImage
	if kmsImage == "" {
		return errors.New("Kms image should not be void")
	}

	// build deployment
	deployment := appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
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
					"node_name": nodeName,
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
					InitContainers: []corev1.Container{
						{
							// TODO update to image in config
							Image: "f4prime/pvinit:v0.0.2",
							Name:  "pvinit",
							Command: []string{
								"python",
								"pvinit.py",
								"--chain",
								chainName,
								"--nodes",
								nodeName,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									MountPath: "/data",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							// TODO update to image in config
							Image: "syncthing/syncthing:1.18",
							// ImagePullPolicy: "Always",
							Name: "syncthing",
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
									SubPath:   "cita-cloud/" + chainName + "/" + nodeName,
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
							Image: networkImage, //"citacloud/network_direct",
							// ImagePullPolicy: "Always",
							Name: "network",
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
									SubPath:   "cita-cloud/" + chainName + "/" + nodeName,
									MountPath: "/data",
								},
								{
									Name:      "network-key",
									MountPath: "/network",
									ReadOnly:  true,
								},
								{
									Name:      "config",
									SubPath:   "network-config",
									MountPath: "/data/network-config.toml",
								},
								{
									Name:      "config",
									SubPath:   "network-log",
									MountPath: "/data/network-log4rs.yaml",
								},
							},
						},
						{
							Image: consensusImage, //"citacloud/consensus_bft",
							// ImagePullPolicy: "Always",
							Name: "consensus",
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
									SubPath:   "cita-cloud/" + chainName + "/" + nodeName,
									MountPath: "/data",
								},
								{
									Name:      "config",
									SubPath:   "consensus-config",
									MountPath: "/data/consensus-config.toml",
								},
								{
									Name:      "config",
									SubPath:   "consensus-log",
									MountPath: "/data/consensus-log4rs.yaml",
								},
								{
									Name:      "config",
									SubPath:   "node_address",
									MountPath: "/data/node_address",
								},
								{
									Name:      "config",
									SubPath:   "node_key",
									MountPath: "/data/node_key",
								},
							},
						},
						{
							Image: executorImage, //"citacloud/executor_evm",
							// ImagePullPolicy: "Always",
							Name: "executor",
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
									SubPath:   "cita-cloud/" + chainName + "/" + nodeName,
									MountPath: "/data",
								},
								{
									Name:      "config",
									SubPath:   "executor-log",
									MountPath: "/data/executor-log4rs.yaml",
								},
							},
						},
						{
							Image: storageImage, //"citacloud/storage_rocksdb",
							// ImagePullPolicy: "Always",
							Name: "storage",
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
								"storage run -p 50003",
							},
							WorkingDir: "/data",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									SubPath:   "cita-cloud/" + chainName + "/" + nodeName,
									MountPath: "/data",
								},
								{
									Name:      "config",
									SubPath:   "storage-log",
									MountPath: "/data/storage-log4rs.yaml",
								},
							},
						},
						{
							Image: controllerImage, //"citacloud/controller",
							// ImagePullPolicy: "Always",
							Name: "controller",
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
									SubPath:   "cita-cloud/" + chainName + "/" + nodeName,
									MountPath: "/data",
								},
								{
									Name:      "config",
									SubPath:   "controller-log",
									MountPath: "/data/controller-log4rs.yaml",
								},
								{
									Name:      "config",
									SubPath:   "controller-config",
									MountPath: "/data/controller-config.toml",
								},
								{
									Name:      "config",
									SubPath:   "genesis",
									MountPath: "/data/genesis.toml",
								},
								{
									Name:      "config",
									SubPath:   "init_sys_config",
									MountPath: "/data/init_sys_config.toml",
								},
								{
									Name:      "config",
									SubPath:   "key_id",
									MountPath: "/data/key_id",
								},
								{
									Name:      "config",
									SubPath:   "node_address",
									MountPath: "/data/node_address",
								},
							},
						},
						{
							Image: kmsImage, //"citacloud/kms_sm",
							// ImagePullPolicy: "Always",
							Name: "kms",
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
								"kms run -p 50005 -k /kms/key_file",
							},
							WorkingDir: "/data",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									SubPath:   "cita-cloud/" + chainName + "/" + nodeName,
									MountPath: "/data",
								},
								{
									Name:      "kms-key",
									MountPath: "/kms",
									ReadOnly:  true,
								},
								{
									Name:      "config",
									SubPath:   "kms-log",
									MountPath: "/data/kms-log4rs.yaml",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "kms-key",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: chainName + "-kms-secret",
								},
							},
						},
						{
							Name: "network-key",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: nodeName + "-network-secret",
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
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: nodeName + "-config",
									},
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

func (r *ChainNodeReconciler) reconcileNodeKmsSecret(
	ctx context.Context,
	chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
	prestartFlag *bool,
) error {
	logger := log.FromContext(ctx)
	// test if the kmsSecret exists
	kmsSecretName := chainConfig.ObjectMeta.Name +
		"-kms-secret"
	var kmsSecret corev1.Secret
	operation := nothingNeeded
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: "default",
		Name:      kmsSecretName,
	}, &kmsSecret); err != nil {
		// can not find Secret
		if apierror.IsNotFound(err) {
			operation = buildNeeded
		} else {
			// other error
			logger.Error(err, "Error finding kms Secret")
			return nil
		}
	} else {
		// Secret found

		// Compare configs
		// changed config               | operation
		// ChainNode.Spec.KmsPassword | update

		// If the operation is updateNeeded, do updates here
		if string(kmsSecret.Data["key_file"]) != chainNode.Spec.KmsPassword {
			operation = updateNeeded
			kmsSecret.Data["key_file"] = []byte(chainNode.Spec.KmsPassword)
		}
	}

	// Do operation
	if operation == buildNeeded {
		*prestartFlag = true
		if errBuild := buildNodeKmsSecret(chainNode, chainConfig, &kmsSecret, kmsSecretName); errBuild != nil {
			logger.Error(errBuild, "Failed building kms Secret")
			return nil
		}

		// The controller of kmsSecret is set to chainConfig since it is a config of the chain instead of a node.
		if errSet := ctrl.SetControllerReference(chainNode, &kmsSecret, r.Scheme); errSet != nil {
			logger.Error(errSet, "set controller reference failed")
			return nil
		}

		if errCreate := r.Create(ctx, &kmsSecret); errCreate != nil {
			logger.Error(errCreate, "Failed create kms Secret")
			return nil
		}

	} else if operation == updateNeeded {
		if errUpdate := r.Update(ctx, &kmsSecret); errUpdate != nil {
			logger.Error(errUpdate, "Update kms Secret failed")
			return nil
		}
	}
	return nil
}

func buildNodeKmsSecret(
	chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
	psecret *corev1.Secret,
	kmsSecretName string) error {
	// init parameters
	secretString := chainNode.Spec.KmsPassword
	if secretString == "" {
		return errors.New("ChainConfig.Spec.KmsPassword should not be empty")
	}

	// build secret
	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kmsSecretName,
			Namespace: "default",
		},
		Type: "Opaque",
		Data: map[string][]byte{
			//"key_file": []byte("MTIzNDU2"),
			"key_file": []byte(secretString),
		},
	}
	secret.DeepCopyInto(psecret)
	return nil
}

func (r *ChainNodeReconciler) reconcileNodeNetworkSecret(
	ctx context.Context,
	chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
	prestartFlag *bool,
) error {
	logger := log.FromContext(ctx)

	// test if the networkSecret exists
	networkSecretName := chainNode.ObjectMeta.Name + "-network-secret"
	var networkSecret corev1.Secret

	operation := nothingNeeded

	if err := r.Get(ctx, types.NamespacedName{
		Namespace: "default",
		Name:      networkSecretName,
	}, &networkSecret); err != nil {
		// can not find Secret
		if apierror.IsNotFound(err) {
			operation = buildNeeded
		} else {
			// other error
			logger.Error(err, "Error finding Network Secret")
			return nil
		}
	} else {
		// Secret found

		// Compare configs
		// changed config            | operation
		// ChainNode.Spec.NetworkKey | update

		// If the operation is updateNeeded, do updates here
		if string(networkSecret.Data["network-key"]) != chainNode.Spec.NetworkKey {
			operation = updateNeeded
			networkSecret.Data["network-key"] = []byte(chainNode.Spec.NetworkKey)
		}
	}

	// Do operation
	if operation == buildNeeded {
		*prestartFlag = true
		if errBuild := buildNodeNetworkSecret(
			chainNode, chainConfig, &networkSecret,
			networkSecretName); errBuild != nil {
			logger.Error(errBuild, "Failed building network Secret")
			return nil
		}

		if errSet := ctrl.SetControllerReference(chainNode, &networkSecret, r.Scheme); errSet != nil {
			logger.Error(errSet, "set controller reference failed")
			return nil
		}

		if errCreate := r.Create(ctx, &networkSecret); errCreate != nil {
			logger.Error(errCreate, "Failed create network Secret")
			return nil
		}

	} else if operation == updateNeeded {
		if errUpdate := r.Update(ctx, &networkSecret); errUpdate != nil {
			logger.Error(errUpdate, "Update network Secret failed")
			return nil
		}
	}
	return nil
}

func buildNodeNetworkSecret(
	chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
	psecret *corev1.Secret,
	networkSecretName string) error {

	// init parameters
	networkKey := chainNode.Spec.NetworkKey

	// build secret
	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkSecretName,
			Namespace: "default",
			// Labels: map[string]string{
			// 	"node_id": nodeID,
			// },
		},
		Type: "Opaque",
		Data: map[string][]byte{
			"network-key": []byte(networkKey), //"MHg2N2ZiMDhlM2NkMThhMGQ2NDVlNGVhZmY0MTY2YmVhMTc4MzIyODJlMjNkMzQxNGJmNzUwYjhjZjQ3YjkzZDQz",
		},
	}
	secret.DeepCopyInto(psecret)
	return nil
}
