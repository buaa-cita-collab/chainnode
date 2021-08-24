package controllers

import (
	"bytes"
	"context"
	citacloudv1 "github.com/buaa-cita/chainnode/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"text/template"
)

func (r *ChainNodeReconciler) reconcileNetworkConfig(
	ctx context.Context,
	chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
) error {
	// network config is mapped into the following files
	// network-config.toml
	// network-log4rs.toml

	logger := log.FromContext(ctx)
	nodeName := chainNode.ObjectMeta.Name
	operation := nothingNeeded

	var networkConfig corev1.ConfigMap
	networkConfigName := nodeName + "-network-config"

	if err := r.Get(ctx, types.NamespacedName{
		Namespace: "default",
		Name:      networkConfigName,
	}, &networkConfig); err != nil {
		// can not find deployment
		if apierror.IsNotFound(err) {
			//build deployment
			operation = buildNeeded
		} else {
			logger.Info("error fetch deployment")
			return err
		}
	} else {

		if chainNode.Status.NodeCount != strconv.Itoa(len(chainConfig.Spec.Nodes)) {
			operation = updateNeeded
		}
	}

	// do operation
	if operation == nothingNeeded {
		return nil
	}

	if err := buildNetworkConfig(chainNode, chainConfig,
		&networkConfig, networkConfigName); err != nil {
		logger.Error(err, "build network config failed")
		return nil
	}

	if operation == buildNeeded {
		if errCreate := r.Create(ctx, &networkConfig); errCreate != nil {
			logger.Error(errCreate, "Create network config failed")
			return nil
		} else {
			logger.Info("Create network config succeed")
		}
	} else if operation == updateNeeded {
		if errUpdate := r.Update(ctx, &networkConfig); errUpdate != nil {
			logger.Error(errUpdate, "Update network config failed")
			return nil
		} else {
			logger.Info("Update network config succeed")
		}
	}
	return nil
}

func buildNetworkConfig(
	chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
	pnetworkConfig *corev1.ConfigMap,
	networkConfigName string,
) error {
	// build network config
	networkConfigTemplateString := `enable_tls = true
port = 40000
{{ range $_, $node := .}}
[[peers]]
ip = "{{ $node }}"
port = 40000
{{ end }}`

	networkLogTemplateString := `# Scan this file for changes every 30 seconds
refresh_rate: 30 seconds

appenders:
  # An appender named "stdout" that writes to stdout
  stdout:
    kind: console

  journey-service:
    kind: rolling_file
    path: "logs/network-service.log"
    policy:
      # Identifies which policy is to be used. If no kind is specified, it will
      # default to "compound".
      kind: compound
      # The remainder of the configuration is passed along to the policy's
      # deserializer, and will vary based on the kind of policy.
      trigger:
        kind: size
        limit: 50mb
      roller:
        kind: fixed_window
        base: 1
        count: 5
        pattern: "logs/network-service.{}.gz"

# Set the default logging level and attach the default appender to the root
root:
  level: info
  appenders:
    - journey-service`
	networkConfigTemplate, err := template.New("networkConfig").Parse(networkConfigTemplateString)
	if err != nil {
		//logger.Error(err,"network config template parse failed")
		// return nil
		return err
	}
	networkLogTemplate, err := template.New("networkLog").Parse(networkLogTemplateString)
	if err != nil {
		// logger.Error(err,"network log template parse failed")
		// return nil
		return err
	}
	networkConfigBuffer := new(bytes.Buffer)
	networkLogBuffer := new(bytes.Buffer)
	nodesIgnoreSelf := make([]string, 0, len(chainConfig.Spec.Nodes))
	for _, node := range chainConfig.Spec.Nodes {
		if node == chainNode.ObjectMeta.Name { // is current node
			continue
		}
		nodesIgnoreSelf = append(nodesIgnoreSelf, node)
	}

	err = networkConfigTemplate.Execute(networkConfigBuffer, nodesIgnoreSelf)
	if err != nil {
		// logger.Error(err,"network config template execute failed")
		// return nil
		return err
	}
	err = networkLogTemplate.Execute(networkLogBuffer, nil)
	if err != nil {
		// logger.Error(err,"network log template execute failed")
		// return nil
		return err
	}

	networkConfig := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkConfigName,
			Namespace: "default",
		},
		Data: map[string]string{
			"network-config": networkConfigBuffer.String(),
			"network-log":    networkLogBuffer.String(),
		},
	}

	networkConfig.DeepCopyInto(pnetworkConfig)
	return nil
}
