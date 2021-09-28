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

	// appv1 "k8s.io/api/apps/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	// corev1 "k8s.io/api/core/v1"
)

// ChainNodeReconciler reconciles a ChainNode object
type ChainNodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=citacloud.buaa.edu.cn,resources=chainnodes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=citacloud.buaa.edu.cn,resources=chainnodes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=citacloud.buaa.edu.cn,resources=chainnodes/finalizers,verbs=update
//+kubebuilder:rbac:groups=citacloud.buaa.edu.cn,resources=chainconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=citacloud.buaa.edu.cn,resources=chainconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ChainNode object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile

func (r *ChainNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("start chainnode reconcile")
	var chainNode citacloudv1.ChainNode

	// fetch chainNode
	if err := r.Get(ctx, req.NamespacedName, &chainNode); err != nil {
		logger.Info("chain node " + req.Name + " has been deleted")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// fetch chainConfig
	var configName, configNamespace string
	if chainNode.Spec.ConfigName == "" {
		// configName not set
		configName = "chainconfig-sample"
	} else {
		configName = chainNode.Spec.ConfigName
	}

	if chainNode.Spec.ConfigNamespace == "" {
		// configNamespace not set
		configNamespace = "default"
	} else {
		configNamespace = chainNode.Spec.ConfigNamespace
	}

	configKey := types.NamespacedName{
		Namespace: configNamespace,
		Name:      configName,
	}
	var chainConfig citacloudv1.ChainConfig
	if err := r.Get(ctx, configKey, &chainConfig); err != nil {
		if apierror.IsNotFound(err) {
			//dont show too much info if it is a not found error
			logger.Info("can not find chainconfig " + configName +
				" in namespace " + configNamespace)
		} else {
			logger.Error(err, "get chainconfig failed")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If chainConfig if not ready, then wait
	// until next run
	if !chainConfig.Status.Ready {
		logger.Info("Waiting chain config to be ready")
		return ctrl.Result{}, nil
	}

	// Checking chainNode update policy
	if chainNode.Spec.UpdatePoilcy != AutoUpdate && chainNode.Spec.UpdatePoilcy != NoUpdate {
		logger.Info("update policy not set or illegal, set to default AutoUpdate")
		chainNode.Spec.UpdatePoilcy = AutoUpdate
		if err := r.Update(ctx, &chainNode); err != nil {
			logger.Error(err, "update chainNode failed")
			return ctrl.Result{}, nil
		}
	}

	restartFlag := false

	if err := r.reconcileConfig(ctx, &chainNode, &chainConfig, &restartFlag); err != nil {
		logger.Error(err, "reconcile config failed")
		return ctrl.Result{}, nil
	}

	if err := r.reconcileNodeKmsSecret(ctx, &chainNode, &chainConfig, &restartFlag); err != nil {
		logger.Error(err, "reconcile kms Secret failed")
		return ctrl.Result{}, nil
	}

	if err := r.reconcileNodeNetworkSecret(ctx, &chainNode, &chainConfig, &restartFlag); err != nil {
		logger.Error(err, "reconcile node network secret failed")
		return ctrl.Result{}, nil
	}

	if err := r.reconcileNodeDeployment(ctx, &chainNode, &chainConfig, &restartFlag); err != nil {
		logger.Error(err, "reconcile node deployment failed")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChainNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
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
				chainConfig := obj.(*citacloudv1.ChainConfig)
				chainName := chainConfig.ObjectMeta.Name
				chainNamespace := chainConfig.ObjectMeta.Namespace
				for _, node := range nodeList.Items {

					// Only append nodes belong to this chain
					if chainNamespace != node.Spec.ConfigNamespace ||
						chainName != node.Spec.ConfigName {
						continue
					}

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
		For(&citacloudv1.ChainNode{}).
		Owns(&appv1.Deployment{}).
		Complete(r)
}
