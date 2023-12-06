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

package defaultcloud

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/kubefin/kubefin/cmd/kubefin-agent/app/options"
	"github.com/kubefin/kubefin/pkg/api"
	"github.com/kubefin/kubefin/pkg/values"
)

const (
	defaultCpuCoreHourlyPrice = 0.08
	defaultRamGBHourlyPrice   = 0.02
	defaultGpuCardHourlyPrice = 12.80
)

type DefaultCloudProvider struct {
	client               kubernetes.Interface
	CpuCoreHourlyPrice   float64
	RamGBHourlyPrice     float64
	GpuCardHourlyPrice   float64
	GPUDeviceLabelKey    string
	GPUDevicePriceList   string
	NodeCPUCoreDeviation string
	NodeRAMGBDeviation   string
}

func NewDefaultCloudProvider(client kubernetes.Interface, agentOptions *options.AgentOptions) (*DefaultCloudProvider, error) {
	var err error

	cpuCoreHourlyPrice, ramGBHourlyPrice, gpuCardHourlyPrice := defaultCpuCoreHourlyPrice, defaultRamGBHourlyPrice, defaultGpuCardHourlyPrice
	if agentOptions.CustomCPUCoreHourPrice != "" {
		cpuCoreHourlyPrice, err = strconv.ParseFloat(agentOptions.CustomCPUCoreHourPrice, 64)
		if err != nil {
			return nil, err
		}
	}

	if agentOptions.CustomRAMGBHourPrice != "" {
		ramGBHourlyPrice, err = strconv.ParseFloat(agentOptions.CustomRAMGBHourPrice, 64)
		if err != nil {
			return nil, err
		}
	}

	if agentOptions.CustomGPUCardHourPrice != "" {
		gpuCardHourlyPrice, err = strconv.ParseFloat(agentOptions.CustomGPUCardHourPrice, 64)
		if err != nil {
			return nil, err
		}
	}

	defaultCloud := DefaultCloudProvider{
		client:               client,
		CpuCoreHourlyPrice:   cpuCoreHourlyPrice,
		RamGBHourlyPrice:     ramGBHourlyPrice,
		GpuCardHourlyPrice:   gpuCardHourlyPrice,
		NodeCPUCoreDeviation: agentOptions.NodeCPUCoreDeviation,
		NodeRAMGBDeviation:   agentOptions.NodeRAMGBDeviation,
		GPUDeviceLabelKey:    agentOptions.CustomGPUDeviceLabelKey,
		GPUDevicePriceList:   agentOptions.CustomGPUDevicePriceList,
	}

	return &defaultCloud, nil
}

func (c *DefaultCloudProvider) ParseClusterInfo(agentOptions *options.AgentOptions) error {
	if agentOptions.ClusterName == "" {
		return fmt.Errorf("please set the cluster name via env CLUSTER_NAME in agent manifest")
	}

	if agentOptions.ClusterId != "" {
		return nil
	}

	systemNS, err := c.client.CoreV1().Namespaces().Get(context.Background(), metav1.NamespaceSystem, metav1.GetOptions{})
	if err != nil {
		return err
	}
	agentOptions.ClusterId = string(systemNS.UID)

	return nil
}

func (c *DefaultCloudProvider) GetNodeHourlyPrice(node *v1.Node) (*api.InstancePriceInfo, error) {
	cpuCoresQuantity := node.Status.Capacity[v1.ResourceCPU]
	ramBytesQuantity := node.Status.Capacity[v1.ResourceMemory]
	gpuCardsQuantity := node.Status.Capacity[values.ResourceGPU]
	cpuCores := cpuCoresQuantity.AsApproximateFloat64()
	ramBytes := ramBytesQuantity.AsApproximateFloat64()
	gpuCards := gpuCardsQuantity.AsApproximateFloat64()
	gpuCardHourlyPrice := c.GpuCardHourlyPrice

	// we can't get real cpu/ram from node.status, take env and add it
	var err error
	CPUCoreDeviation := 0.0
	if c.NodeCPUCoreDeviation != "" {
		CPUCoreDeviation, err = strconv.ParseFloat(c.NodeCPUCoreDeviation, 64)
		if err != nil {
			klog.Errorf("Parse %s error:%v", c.NodeCPUCoreDeviation, err)
			return nil, err
		}
	}

	RAMGBDeviation := 0.0
	if c.NodeRAMGBDeviation != "" {
		RAMGBDeviation, err = strconv.ParseFloat(c.NodeRAMGBDeviation, 64)
		if err != nil {
			klog.Errorf("Parse %s error:%v", c.NodeRAMGBDeviation, err)
			return nil, err
		}
	}

	if c.GPUDeviceLabelKey != "" && c.GPUDevicePriceList != "" {
		priceList, err := parseGPUDevicePriceList(c.GPUDevicePriceList)
		if err != nil {
			klog.Errorf("Parse %s error:%v", c.GPUDevicePriceList, err)
			return nil, err
		}

		deviceName, labelExists := node.Labels[c.GPUDeviceLabelKey]
		if labelExists {
			if price, exists := priceList[deviceName]; exists {
				gpuCardHourlyPrice = price
			} else {
				klog.V(4).Infof("Unknown device price, use generic GPU card hourly price")
			}
		} else {
			klog.V(4).Info("Unknown device, use generic GPU card hourly price")
		}
	}

	return &api.InstancePriceInfo{
		NodeTotalHourlyPrice: c.CpuCoreHourlyPrice*cpuCores + c.RamGBHourlyPrice*(ramBytes/values.GBInBytes) + c.GpuCardHourlyPrice*gpuCards,
		CPUCore:              cpuCores + CPUCoreDeviation,
		CPUCoreHourlyPrice:   c.CpuCoreHourlyPrice,
		RamGiB:               (ramBytes / values.GBInBytes) + RAMGBDeviation,
		RAMGiBHourlyPrice:    c.RamGBHourlyPrice,
		GPUCards:             gpuCards,
		GPUCardHourlyPrice:   gpuCardHourlyPrice,
		InstanceType:         "default_instance_type",
		BillingMode:          values.BillingModeOnDemand,
		BillingPeriod:        0,
		Region:               "default_region",
		CloudProvider:        api.CloudProviderOnPremise,
	}, nil
}

// Example T1:12;A100:15
func parseGPUDevicePriceList(priceList string) (map[string]float64, error) {
	deviceItems := strings.Split(priceList, ";")
	ret := make(map[string]float64, len(deviceItems))
	for _, deviceItem := range deviceItems {
		splitDeviceItem := strings.Split(deviceItem, ":")
		if len(splitDeviceItem) != 2 {
			return nil, fmt.Errorf("failed to parse GPU device price list")
		}
		deviceName, price := splitDeviceItem[0], splitDeviceItem[1]
		priceAsFloat, err := strconv.ParseFloat(price, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse price of %v", deviceName)
		}
		ret[deviceName] = priceAsFloat
	}

	return ret, nil
}
