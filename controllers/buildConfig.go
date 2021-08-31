package controllers

import (
	"context"
	"errors"
	citacloudv1 "github.com/buaa-cita/chainnode/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	// "strconv"
)

func (r *ChainNodeReconciler) reconcileConfig(
	ctx context.Context,
	chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
	prestartFlag *bool,
) error {
	logger := log.FromContext(ctx)
	nodeName := chainNode.ObjectMeta.Name
	operation := nothingNeeded

	var config corev1.ConfigMap
	configName := nodeName + "-config"

	// checking config to make sure operation needed
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: "default",
		Name:      configName,
	}, &config); err != nil {
		// can not find configmap
		if apierror.IsNotFound(err) {
			operation = buildNeeded
			chainNode.Status.LogLevel = chainNode.Spec.LogLevel
			if chainNode.Spec.UpdatePoilcy == "AutoUpdate" ||
				len(chainNode.Status.Nodes) == 0 {
				chainNode.Status.Nodes = make([]string, len(chainConfig.Spec.Nodes))
				copy(chainNode.Status.Nodes, chainConfig.Spec.Nodes)
			}
		} else {
			logger.Info("Error fetch configmap")
			return err
		}
	} else {
		if chainNode.Spec.LogLevel != chainNode.Status.LogLevel {
			operation = updateNeeded
			chainNode.Status.LogLevel = chainNode.Spec.LogLevel
		} else if chainNode.Spec.UpdatePoilcy == AutoUpdate &&
			len(chainNode.Status.Nodes) != len(chainConfig.Spec.Nodes) {
			operation = updateNeeded
			chainNode.Status.Nodes = make([]string, len(chainConfig.Spec.Nodes))
			copy(chainNode.Status.Nodes, chainConfig.Spec.Nodes)
		}
	}

	// do operation
	if operation == nothingNeeded {
		return nil
	}

	*prestartFlag = true

	if err := buildConfig(chainNode, chainConfig,
		&config, configName, logger); err != nil {
		logger.Error(err, "Build configmap failed")
		return nil
	}

	if operation == buildNeeded {
		if errCreate := r.Create(ctx, &config); errCreate != nil {
			logger.Error(errCreate, "Create network config failed")
			return nil
		} else {
			logger.Info("Create configmap succeed")
		}
	} else if operation == updateNeeded {
		if errUpdate := r.Update(ctx, &config); errUpdate != nil {
			logger.Error(errUpdate, "Update configmap failed")
			return nil
		} else {
			logger.Info("Update configmap succeed")
		}
	}

	// Writing back status
	if err := r.Status().Update(ctx, chainNode); err != nil {
		logger.Error(err, "Writing back status failed")
	}

	return nil
}

func buildConfig(
	chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
	pconfig *corev1.ConfigMap,
	configName string,
	logger logr.Logger,
) error {
	// build the configmap and copy into pconfig

	// No config is related to consensus-config.toml
	consensusConfig := `network_port = 50000
controller_port = 50004
node_id = 0`

	consensusLog, err := genLogConfig(chainNode, chainConfig, "consensus")
	if err != nil {
		logger.Error(err, "Building consensus log config failed")
		return nil
	}

	// No config is related to consensus-config.toml
	controllerConfig := `network_port = 50000
consensus_port = 50001
storage_port = 50003
kms_port = 50005
executor_port = 50002
block_delay_number = 0`

	controllerLog, err := genLogConfig(chainNode, chainConfig, "controller")
	if err != nil {
		logger.Error(err, "Building controller log config failed")
		return nil
	}

	executorLog, err := genLogConfig(chainNode, chainConfig, "executor")
	if err != nil {
		logger.Error(err, "Building executor log config failed")
		return nil
	}

	genesisConfig, err := genGenesisConfig(chainNode, chainConfig)
	if err != nil {
		logger.Error(err, "Building genesis config failed")
		return nil
	}

	initSysConfig, err := genInitSysConfig(chainNode, chainConfig)
	if err != nil {
		logger.Error(err, "Building init sys config failed")
		return nil
	}

	// key_id is not configurable
	keyID := "0"

	kmsLog, err := genLogConfig(chainNode, chainConfig, "kms")
	if err != nil {
		logger.Error(err, "Building kms log config failed")
		return nil
	}

	nodeAddress := chainNode.Spec.NodeAddress
	if nodeAddress == "" {
		return errors.New("node address should not be empty")
	}

	nodeKey := chainNode.Spec.NodeKey
	if nodeKey == "" {
		return errors.New("node key should not be empty")
	}

	networkConfig, err := genNetworkConfig(chainNode, chainConfig)
	if err != nil {
		logger.Error(err, "Building network config failed")
		return nil
	}

	networkLog, err := genLogConfig(chainNode, chainConfig, "network")
	if err != nil {
		logger.Error(err, "Building network log config failed")
		return nil
	}

	storageLog, err := genLogConfig(chainNode, chainConfig, "storage")
	if err != nil {
		logger.Error(err, "Building storage log config failed")
		return nil
	}

	config := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configName,
			Namespace: "default",
		},
		Data: map[string]string{
			"consensus-config":  consensusConfig,
			"consensus-log":     consensusLog,
			"controller-config": controllerConfig,
			"controller-log":    controllerLog,
			"executor-log":      executorLog,
			"genesis":           genesisConfig,
			"init_sys_config":   initSysConfig,
			"key_id":            keyID,
			"kms-log":           kmsLog,
			"network-config":    networkConfig,
			"network-log":       networkLog,
			"node_address":      nodeAddress,
			"node_key":          nodeKey,
			"storage-log":       storageLog,
		},
	}
	config.DeepCopyInto(pconfig)
	return nil
}

// generate gensis.toml
func genGenesisConfig(
	chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
) (string, error) {

	data := map[string]string{
		"timestamp": chainConfig.Spec.Timestamp,
		"prevhash":  chainConfig.Spec.PrevHash,
	}

	templateString := `timestamp = {{.timestamp}}
prevhash = "{{.prevhash}}"`

	return templateBuilder(templateString, data)
}

type initSysData struct {
	ChainID       string
	Validators    []string
	Admin         string
	BlockInterval string
}

// generate init_sys_config.toml
func genInitSysConfig(
	chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
) (string, error) {

	data := initSysData{
		ChainID:       sha3_256HexString(chainConfig.ObjectMeta.Name),
		Validators:    chainConfig.Spec.Authorities,
		Admin:         chainConfig.Spec.SuperAdmin,
		BlockInterval: chainConfig.Spec.BlockInterval,
	}

	templateString := `version = 0
chain_id = "{{.ChainID}}"
admin = "{{.Admin}}"
block_interval = {{.BlockInterval}}
validators = [{{range $_,$v := .Validators}}"{{$v}}", {{end}}]`
	return templateBuilder(templateString, data)
}

// generate network-config.toml
func genNetworkConfig(
	chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
) (string, error) {
	templateString := `enable_tls = true
port = 40000
{{ range $_, $node := .}}
[[peers]]
ip = "{{ $node }}"
port = 40000
{{ end }}`

	nodesIgnoreSelf := make([]string, 0, len(chainNode.Status.Nodes))
	for _, node := range chainNode.Status.Nodes {
		if node == chainNode.ObjectMeta.Name { // is current node
			continue
		}
		nodesIgnoreSelf = append(nodesIgnoreSelf, node)
	}

	return templateBuilder(templateString, nodesIgnoreSelf)
}

// consensus-log4rs.yaml
// controller-log4rs.yaml
// executor-log4rs.yaml
// kms-log4rs.yaml
// network-log4rs.yaml
// storage-log4rs.yaml
func genLogConfig(
	chainNode *citacloudv1.ChainNode,
	chainConfig *citacloudv1.ChainConfig,
	serviceName string,
) (string, error) {

	level := "info"
	if chainNode.Spec.LogLevel != "" {
		level = chainNode.Spec.LogLevel
	}

	configs := map[string]string{
		"serviceName": serviceName,
		"level":       level,
	}

	templateString := `# Scan this file for changes every 30 seconds
refresh_rate: 30 seconds

appenders:
  # An appender named "stdout" that writes to stdout
  stdout:
    kind: console

  journey-service:
    kind: rolling_file
    path: "logs/{{.serviceName}}-service.log"
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
        pattern: "logs/{{.serviceName}}-service.{}.gz"

# Set the default logging level and attach the default appender to the root
root:
  level: {{.level}}
  appenders:
    - journey-service`

	return templateBuilder(templateString, configs)
}
