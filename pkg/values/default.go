/*
Copyright 2022 The KubeFin Authors

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

package values

import (
	"time"

	v1 "k8s.io/api/core/v1"
)

const (
	KubeFinNamespace = "kubefin"
	KubeFinAgentName = "kubefin-agent"

	// DefaultLeaseDuration is the LeaseDuration for leader election.
	DefaultLeaseDuration = 15 * time.Second
	// DefaultRenewDeadline is the RenewDeadline for leader election.
	DefaultRenewDeadline = 10 * time.Second
	// DefaultRetryPeriod is the RetryPeriod for leader election.
	DefaultRetryPeriod = 5 * time.Second
	// DefaultMetricsQueryPeriod is the period to query usage metrics.
	DefaultMetricsQueryPeriod = 15 * time.Second

	LostConnectionTimeoutThreshold = time.Minute * 3 / time.Second

	GBInBytes              = 1024.0 * 1024.0 * 1024.0
	CoreInMCore            = 1000.0
	HourInSeconds          = 3600.0
	MetricsPeriodInSeconds = 15.0

	CloudProviderEnv          = "CLOUD_PROVIDER"
	ClusterNameEnv            = "CLUSTER_NAME"
	ClusterIdEnv              = "CLUSTER_ID"
	LeaderElectionIDEnv       = "LEADER_ELECTION_ID"
	QueryBackendEndpointEnv   = "QUERY_BACKEND_ENDPOINT"
	NodeCPUDeviationEnv       = "NODE_CPU_DEVIATION"
	NodeRAMDeviationEnv       = "NODE_RAM_DEVIATION"
	CPUMemoryCostRatioEnv     = "CPUCORE_RAMGB_PRICE_RATIO"
	GPUCPUCostRatioEnv        = "GPUAMOUNT_CPUCORE_PRICE_RATIO"
	CustomCPUCoreHourPriceEnv = "CUSTOM_CPU_CORE_HOUR_PRICE"
	CustomRAMGBHourPriceEnv   = "CUSTOM_RAM_GB_HOUR_PRICE"

	// CustomGPUCardHourPriceEnv indicates price per hour for an nvidia GPU card,
	// and it does not distinguish the specific device of the GPU card.
	// Because Kubernetes treats extended resources such as GPU as scalar resources by default.
	// Whether it is T4 or A100, it's regarded as one "nvidia.com/gpu" resource.
	CustomGPUCardHourPriceEnv = "CUSTOM_GPU_CARD_HOUR_PRICE"
	// CustomGPUDeviceLabelKeyEnv indicates the key of label which indicates the type of GPUs.
	// If different nodes in the cluster have different types of GPUs,
	// Kubernetes recommends using Node Labels and Node Selectors to schedule pods to appropriate nodes.
	// See https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus/#clusters-containing-different-types-of-gpus.
	// Kubefin supports using user-specified labels to identify specific GPU devices and calculate their prices.
	// You can also use GPU feature discovery to implement automatic node labelling.
	// See details in https://github.com/NVIDIA/gpu-feature-discovery/blob/main/README.md.
	CustomGPUDeviceLabelKeyEnv = "CUSTOM_GPU_DEVICE_LABEL_KEY"
	// CustomGPUDevicePriceListEnv indicates price list of different types of GPUs.
	// It will use ";" to separate different device items and use ":" to separate device name and price per hour.
	// CustomGPUDevicePriceListEnv needs to be used together with CustomGPUDeviceLabelKeyEnv.
	// Example:
	// A100:12;T4:15
	CustomGPUDevicePriceListEnv = "CUSTOM_GPU_DEVICE_PRICE_LIST"

	MultiTenantHeader       = "X-Scope-OrgID"
	ClusterIdQueryParameter = "cluster_id"

	DefaultStepSeconds = 3600
	// DefaultDetailStepSeconds is used to show the fine-grained line chart of cpu/memory data
	DefaultDetailStepSeconds = 600

	CloudProviderACK       = "ack"
	CloudProviderEKS       = "eks"
	CloudProviderOnPremise = "onpremise"
	CloudProviderAuto      = "auto"

	ResourceGPU = v1.ResourceName("nvidia.com/gpu")
)
