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
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	citacloudv1 "github.com/buaa-cita/chainnode/api/v1"
	"github.com/go-logr/logr"
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
		logger.Info("chain config " + req.Name + " has been deleted")
		//logger.Error(err, "get ChainConfig failed")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	changed := false
	statusChanged := false

	// Setting default values including:
	// Field              Value
	// BlockInterval      "3"
	// Timestamp          unix_now*1000(string)
	// PrevHash            "0x0000000000000000000000000000000000000000000000000000000000000000"
	// EnableTLS          True
	if chainConfig.Spec.BlockInterval == "" &&
		chainConfig.Status.BlockInterval == "" {
		chainConfig.Spec.BlockInterval = "3"
		changed = true
	}
	if chainConfig.Spec.Timestamp == "" &&
		chainConfig.Status.Timestamp == "" {
		ts := strconv.FormatInt(time.Now().Unix()*1000, 10)
		chainConfig.Spec.Timestamp = ts
		changed = true
	}
	if chainConfig.Spec.PrevHash == "" &&
		chainConfig.Status.PrevHash == "" {
		chainConfig.Spec.PrevHash = "0x0000000000000000000000000000000000000000000000000000000000000000"
		changed = true
	}
	if chainConfig.Spec.EnableTLS == "" &&
		chainConfig.Status.EnableTLS == "" {
		chainConfig.Spec.EnableTLS = "true"
		changed = true
	}

	// Set backup or restore unchangeable value from backup
	setRestoreBackup(&chainConfig, logger, &changed,
		&statusChanged)

	// chainConfig.Status.Ready indicates that default
	// value settings has been done before so it does
	// not needs to be done this time.
	if !chainConfig.Status.Ready {
		// Set ready flag
		logger.Info("status ready")
		chainConfig.Status.Ready = true
		statusChanged = true
	}
	// And write back
	if changed {
		if err := r.Update(ctx, &chainConfig); err != nil {
			logger.Error(err, "update chain config failed")
			return ctrl.Result{}, nil
		}
	} else {
		logger.Info("update chain config succeed")
	}

	// verify
	// var chainC citacloudv1.ChainConfig
	// if err := r.Get(ctx, req.NamespacedName, &chainC); err != nil {
	// 	logger.Error(err, "re get chain config failed")
	// 	return ctrl.Result{}, nil
	// } else {
	// 	logger.Info("verify", "after write", chainConfig.Status.Ready, "read", chainC.Status.Ready)
	// }

	// And write status back
	if statusChanged {
		if err := r.Status().Update(ctx, &chainConfig); err != nil {
			logger.Error(err, "update chain config status failed")
			return ctrl.Result{}, nil
		}
	} else {
		logger.Info("update chain config status succeed")
	}

	// verify
	// if err := r.Get(ctx, req.NamespacedName, &chainC); err != nil {
	// 	logger.Error(err, "re get chain config failed")
	// 	return ctrl.Result{}, nil
	// } else {
	// 	logger.Info("verify", "after write status", chainConfig.Status.Ready, "read", chainC.Status.Ready)
	// }

	return ctrl.Result{}, nil
}

func setRestoreBackup(
	chainConfig *citacloudv1.ChainConfig,
	logger logr.Logger,
	pchanged *bool,
	pStatusChanged *bool,
) {
	// Write to Status to backup if backup is not set
	// Compare to Status to restore some fields that
	// should be not *pchanged
	if len(chainConfig.Status.Authorities) == 0 {
		chainConfig.Status.Authorities = make([]string, len(chainConfig.Spec.Authorities))
		copy(chainConfig.Status.Authorities, chainConfig.Spec.Authorities)
		*pStatusChanged = true
	} else if !stringSliceEqual(chainConfig.Spec.Authorities,
		chainConfig.Status.Authorities) {
		logger.Info("Authorities change detected, new values is " +
			fmt.Sprint(chainConfig.Spec.Authorities) +
			", restored to " +
			fmt.Sprint(chainConfig.Status.Authorities),
		)
		chainConfig.Spec.Authorities = make([]string, len(chainConfig.Status.Authorities))
		copy(chainConfig.Spec.Authorities, chainConfig.Status.Authorities)
		*pchanged = true
	}
	if chainConfig.Status.SuperAdmin == "" {
		chainConfig.Status.SuperAdmin = chainConfig.Spec.SuperAdmin
		*pStatusChanged = true
	} else if chainConfig.Spec.SuperAdmin != chainConfig.Status.SuperAdmin {
		logger.Info("SuperAdmin change detected, new values is " +
			fmt.Sprint(chainConfig.Spec.SuperAdmin) +
			", restored to " +
			fmt.Sprint(chainConfig.Status.SuperAdmin),
		)
		chainConfig.Spec.SuperAdmin = chainConfig.Status.SuperAdmin
		*pchanged = true
	}
	if chainConfig.Status.BlockInterval == "" {
		chainConfig.Status.BlockInterval = chainConfig.Spec.BlockInterval
		*pStatusChanged = true
	} else if chainConfig.Spec.BlockInterval != chainConfig.Status.BlockInterval {
		logger.Info("BlockInterval change detected, new values is " +
			fmt.Sprint(chainConfig.Spec.BlockInterval) +
			", restored to " +
			fmt.Sprint(chainConfig.Status.BlockInterval),
		)
		chainConfig.Spec.BlockInterval = chainConfig.Status.BlockInterval
		*pchanged = true
	}
	if chainConfig.Status.Timestamp == "" {
		chainConfig.Status.Timestamp = chainConfig.Spec.Timestamp
		*pStatusChanged = true
	} else if chainConfig.Spec.Timestamp != chainConfig.Status.Timestamp {
		logger.Info("Timestamp change detected, new values is " +
			fmt.Sprint(chainConfig.Spec.Timestamp) +
			", restored to " +
			fmt.Sprint(chainConfig.Status.Timestamp),
		)
		chainConfig.Spec.Timestamp = chainConfig.Status.Timestamp
		*pchanged = true
	}
	if chainConfig.Status.PrevHash == "" {
		chainConfig.Status.PrevHash = chainConfig.Spec.PrevHash
		*pStatusChanged = true
	} else if chainConfig.Spec.PrevHash != chainConfig.Status.PrevHash {
		logger.Info("PrevHash change detected, new values is " +
			fmt.Sprint(chainConfig.Spec.PrevHash) +
			", restored to " +
			fmt.Sprint(chainConfig.Status.PrevHash),
		)
		chainConfig.Spec.PrevHash = chainConfig.Status.PrevHash
		*pchanged = true
	}
	if chainConfig.Status.EnableTLS == "" {
		chainConfig.Status.EnableTLS = chainConfig.Spec.EnableTLS
		*pStatusChanged = true
	} else if chainConfig.Spec.EnableTLS != chainConfig.Status.EnableTLS {
		logger.Info("EnableTLS change detected, new values is " +
			fmt.Sprint(chainConfig.Spec.EnableTLS) +
			", restored to " +
			fmt.Sprint(chainConfig.Status.EnableTLS),
		)
		chainConfig.Spec.EnableTLS = chainConfig.Status.EnableTLS
		*pchanged = true
	}
	if chainConfig.Status.NetworkImage == "" {
		chainConfig.Status.NetworkImage = chainConfig.Spec.NetworkImage
		*pStatusChanged = true
	} else if chainConfig.Spec.NetworkImage != chainConfig.Status.NetworkImage {
		logger.Info("NetworkImage change detected, new values is " +
			fmt.Sprint(chainConfig.Spec.NetworkImage) +
			", restored to " +
			fmt.Sprint(chainConfig.Status.NetworkImage),
		)
		chainConfig.Spec.NetworkImage = chainConfig.Status.NetworkImage
		*pchanged = true
	}
	if chainConfig.Status.ConsensusImage == "" {
		chainConfig.Status.ConsensusImage = chainConfig.Spec.ConsensusImage
		*pStatusChanged = true
	} else if chainConfig.Spec.ConsensusImage != chainConfig.Status.ConsensusImage {
		logger.Info("ConsensusImage change detected, new values is " +
			fmt.Sprint(chainConfig.Spec.ConsensusImage) +
			", restored to " +
			fmt.Sprint(chainConfig.Status.ConsensusImage),
		)
		chainConfig.Spec.ConsensusImage = chainConfig.Status.ConsensusImage
		*pchanged = true
	}
	if chainConfig.Status.ExecutorImage == "" {
		chainConfig.Status.ExecutorImage = chainConfig.Spec.ExecutorImage
		*pStatusChanged = true
	} else if chainConfig.Spec.ExecutorImage != chainConfig.Status.ExecutorImage {
		logger.Info("ExecutorImage change detected, new values is " +
			fmt.Sprint(chainConfig.Spec.ExecutorImage) +
			", restored to " +
			fmt.Sprint(chainConfig.Status.ExecutorImage),
		)
		chainConfig.Spec.ExecutorImage = chainConfig.Status.ExecutorImage
		*pchanged = true
	}
	if chainConfig.Status.StorageImage == "" {
		chainConfig.Status.StorageImage = chainConfig.Spec.StorageImage
		*pStatusChanged = true
	} else if chainConfig.Spec.StorageImage != chainConfig.Status.StorageImage {
		logger.Info("StorageImage change detected, new values is " +
			fmt.Sprint(chainConfig.Spec.StorageImage) +
			", restored to " +
			fmt.Sprint(chainConfig.Status.StorageImage),
		)
		chainConfig.Spec.StorageImage = chainConfig.Status.StorageImage
		*pchanged = true
	}
	if chainConfig.Status.ControllerImage == "" {
		chainConfig.Status.ControllerImage = chainConfig.Spec.ControllerImage
		*pStatusChanged = true
	} else if chainConfig.Spec.ControllerImage != chainConfig.Status.ControllerImage {
		logger.Info("ControllerImage change detected, new values is " +
			fmt.Sprint(chainConfig.Spec.ControllerImage) +
			", restored to " +
			fmt.Sprint(chainConfig.Status.ControllerImage),
		)
		chainConfig.Spec.ControllerImage = chainConfig.Status.ControllerImage
		*pchanged = true
	}
	if chainConfig.Status.KmsImage == "" {
		chainConfig.Status.KmsImage = chainConfig.Spec.KmsImage
		*pStatusChanged = true
	} else if chainConfig.Spec.KmsImage != chainConfig.Status.KmsImage {
		logger.Info("KmsImage change detected, new values is " +
			fmt.Sprint(chainConfig.Spec.KmsImage) +
			", restored to " +
			fmt.Sprint(chainConfig.Status.KmsImage),
		)
		chainConfig.Spec.KmsImage = chainConfig.Status.KmsImage
		*pchanged = true
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChainConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&citacloudv1.ChainConfig{}).
		Complete(r)
}
