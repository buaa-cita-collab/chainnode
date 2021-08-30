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
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	citacloudv1 "github.com/buaa-cita/chainnode/api/v1"
)

// ChainConfigReconciler reconciles a ChainConfig object
type ChainConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=citacloud.buaa.edu.cn,resources=chainconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=citacloud.buaa.edu.cn,resources=chainconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=citacloud.buaa.edu.cn,resources=chainconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ChainConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile

func (r *ChainConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// your logic here
	logger := log.FromContext(ctx)
	logger.Info("start chainconfig reconcile")
	var chainConfig citacloudv1.ChainConfig

	// Fetching chainConfig
	if err := r.Get(ctx, req.NamespacedName, &chainConfig); err != nil {
		logger.Error(err, "get ChainConfig failed")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// chainConfig.Status.Ready indicates that default
	// value settings has been done before
	if chainConfig.Status.Ready {
		return ctrl.Result{}, nil
	}

	// Setting default values including:
	// Field              Value
	// BlockInterval      "3"
	// Timestamp          unix_now*1000(string)
	// PrevHash            "0x0000000000000000000000000000000000000000000000000000000000000000"
	// EnableTLS          True
	if chainConfig.Spec.BlockInterval == "" {
		chainConfig.Spec.BlockInterval = "3"
	}
	if chainConfig.Spec.Timestamp == "" {
		ts := strconv.FormatInt(time.Now().Unix()*1000, 10)
		chainConfig.Spec.Timestamp = ts
	}
	if chainConfig.Spec.PrevHash == "" {
		chainConfig.Spec.PrevHash = "0x0000000000000000000000000000000000000000000000000000000000000000"
	}
	if chainConfig.Spec.EnableTLS == "" {
		chainConfig.Spec.Timestamp = "true"
	}

	// Set ready flag
	chainConfig.Status.Ready = true

	// And write back
	if err := r.Update(ctx, &chainConfig); err != nil {
		logger.Error(err, "update chain config failed")
		return ctrl.Result{}, nil
	}

	logger.Info("chainconfig status ready", "ready:", chainConfig.Status.Ready)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChainConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&citacloudv1.ChainConfig{}).
		Complete(r)
}
