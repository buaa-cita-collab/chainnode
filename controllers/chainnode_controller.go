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

    appv1 "k8s.io/api/apps/v1"
	citacloudv1 "github.com/buaa-cita/chainnode/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	apierror "k8s.io/apimachinery/pkg/api/errors"

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
func (r *ChainNodeReconciler) Reconcile(ctx context.Context,
         req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
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
	configKey:=types.NamespacedName{
		Namespace: req.NamespacedName.Namespace,
		Name:configName}
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

	// 
	deploymentName := req.Name+"_deployment"
	var deployment appv1.Deployment
	if err:=r.Get(ctx,types.NamespacedName{
		Namespace:req.Namespace,
		Name:deploymentName,
	},&deployment); err!=nil {
		// can not find deployment 
		if apierror.IsNotFound(err) {
		    //build deployment
			if errBuild:=buildNodeDeployment(chainNode,chainConfig,&deployment);
					errBuild!=nil {
				logger.Error(err,"Failed building node Deployment")
				// return nil to avoid rerun reconcile
				return ctrl.Result{},nil 
			}
		} else {
			// other error
			logger.Error(err, "can not find deployment")
			return ctrl.Result{},client.IgnoreNotFound(err) 
		}
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
	deployment *appv1.Deployment) error {
		
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
