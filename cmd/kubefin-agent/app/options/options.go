/*
Copyright 2022 The KubeFin Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicabl e law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package options

import (
	"os"
	"time"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appv1 "k8s.io/client-go/listers/apps/v1"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	baseconfig "k8s.io/component-base/config"

	insightlister "kubefin.dev/kubefin/pkg/generated/listers/insight/v1alpha1"
	"kubefin.dev/kubefin/pkg/values"
)

type AgentOptions struct {
	LeaderElection       baseconfig.LeaderElectionConfiguration
	ScrapMetricsInterval time.Duration
	LeaderElectionID     string

	CloudProvider string
	ClusterName   string
	ClusterId     string

	NodeCPUCoreDeviation     string
	NodeRAMGBDeviation       string
	CPUMemoryCostRatio       string
	GPUCPUCostRatio          string
	CustomCPUCoreHourPrice   string
	CustomRAMGBHourPrice     string
	CustomGPUCardHourPrice   string
	CustomGPUDeviceLabelKey  string
	CustomGPUDevicePriceList string
}

type CoreResourceInformerLister struct {
	NodeInformer              cache.SharedIndexInformer
	NamespaceInformer         cache.SharedIndexInformer
	PodInformer               cache.SharedIndexInformer
	DeploymentInformer        cache.SharedIndexInformer
	StatefulSetInformer       cache.SharedIndexInformer
	DaemonSetInformer         cache.SharedIndexInformer
	CustomWorkloadCfgInformer cache.SharedIndexInformer
	NodeLister                v1.NodeLister
	PodLister                 v1.PodLister
	DeploymentLister          appv1.DeploymentLister
	StatefulSetLister         appv1.StatefulSetLister
	DaemonSetLister           appv1.DaemonSetLister
	CustomWorkloadCfgLister   insightlister.CustomAllocationConfigurationLister
}

// NewAgentOptions builds an empty options.
func NewAgentOptions() *AgentOptions {
	return &AgentOptions{
		LeaderElection: baseconfig.LeaderElectionConfiguration{
			ResourceLock:      resourcelock.LeasesResourceLock,
			ResourceNamespace: values.KubeFinNamespace,
			ResourceName:      values.KubeFinAgentName,
			LeaseDuration:     metav1.Duration{Duration: values.DefaultLeaseDuration},
			RenewDeadline:     metav1.Duration{Duration: values.DefaultRenewDeadline},
			RetryPeriod:       metav1.Duration{Duration: values.DefaultRetryPeriod},
		},
		ScrapMetricsInterval:     values.DefaultMetricsQueryPeriod,
		LeaderElectionID:         os.Getenv(values.LeaderElectionIDEnv),
		CloudProvider:            os.Getenv(values.CloudProviderEnv),
		ClusterName:              os.Getenv(values.ClusterNameEnv),
		ClusterId:                os.Getenv(values.ClusterIdEnv),
		CPUMemoryCostRatio:       os.Getenv(values.CPUMemoryCostRatioEnv),
		GPUCPUCostRatio:          os.Getenv(values.GPUCPUCostRatioEnv),
		CustomCPUCoreHourPrice:   os.Getenv(values.CustomCPUCoreHourPriceEnv),
		CustomRAMGBHourPrice:     os.Getenv(values.CustomRAMGBHourPriceEnv),
		CustomGPUCardHourPrice:   os.Getenv(values.CustomGPUCardHourPriceEnv),
		CustomGPUDeviceLabelKey:  os.Getenv(values.CustomGPUDeviceLabelKeyEnv),
		CustomGPUDevicePriceList: os.Getenv(values.CustomGPUDevicePriceListEnv),
		NodeCPUCoreDeviation:     os.Getenv(values.NodeCPUDeviationEnv),
		NodeRAMGBDeviation:       os.Getenv(values.NodeRAMDeviationEnv),
	}
}

func (o *AgentOptions) Complete() error {
	return nil
}

func (o *AgentOptions) Validate() error {
	return nil
}

func (o *AgentOptions) ApplyTo() {
}

func (o *AgentOptions) AddFlags(flags *pflag.FlagSet) {
}
