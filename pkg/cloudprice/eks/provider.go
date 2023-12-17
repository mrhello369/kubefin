/*
Copyright 2023 The KubeFin Authors

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

package eks

import (
	"context"
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"kubefin.dev/go-tools/awspricing"
	"kubefin.dev/kubefin/cmd/kubefin-agent/app/options"
	"kubefin.dev/kubefin/pkg/api"
	cloudpriceapis "kubefin.dev/kubefin/pkg/cloudprice/apis"
	"kubefin.dev/kubefin/pkg/values"
)

const (
	eksNodeRegionLabelKey = "topology.kubernetes.io/region"
	eksNodeTypeLabelKey   = "node.kubernetes.io/instance-type"
)

type EKSCloudProvider struct {
	client           kubernetes.Interface
	awsPricingClient *awspricing.AWSEC2PriceClient

	cpuMemoryCostRatio float64
	region             string
}

func NewEKSCloudProvider(client kubernetes.Interface, agentOptions *options.AgentOptions) (*EKSCloudProvider, error) {
	var region string
	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, node := range nodes.Items {
		labels := node.Labels
		if len(labels) == 0 {
			continue
		}
		if rg, ok := labels[eksNodeRegionLabelKey]; ok {
			region = rg
			break
		}
	}
	if region == "" {
		return nil, fmt.Errorf("could not find region")
	}
	awsPricingClient, err := awspricing.NewAWSEC2PriceClient(region)
	if err != nil {
		return nil, err
	}

	cpuMemoryCostRatio := cloudpriceapis.DefaultCPUMemoryCostRatio
	if agentOptions.CPUMemoryCostRatio != "" {
		cpuMemoryCostRatio, err = strconv.ParseFloat(agentOptions.CPUMemoryCostRatio, 64)
		if err != nil {
			return nil, err
		}
	}
	eksCloud := EKSCloudProvider{
		client:             client,
		awsPricingClient:   awsPricingClient,
		cpuMemoryCostRatio: cpuMemoryCostRatio,
		region:             region,
	}

	systemNS, err := client.CoreV1().Namespaces().Get(context.Background(), metav1.NamespaceSystem, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	agentOptions.ClusterId = string(systemNS.UID)

	return &eksCloud, nil
}

func (e *EKSCloudProvider) Start(ctx context.Context) {
	go e.awsPricingClient.Start()
}

func (e *EKSCloudProvider) GetNodeHourlyPrice(node *v1.Node) (*api.InstancePriceInfo, error) {
	if node.Labels == nil {
		return nil, fmt.Errorf("node(%s) has no labels", node.Name)
	}

	nodeType, ok := node.Labels[eksNodeTypeLabelKey]
	if !ok {
		return nil, fmt.Errorf("node(%s) has no label %s", node.Name, eksNodeRegionLabelKey)
	}

	nodePriceInfo, err := e.awsPricingClient.GetOnDemandEC2PriceInfo(nodeType)
	if err != nil {
		return nil, err
	}

	var price float64
	if nodePriceInfo.PricePerUnit.USD != "" {
		price, err = strconv.ParseFloat(nodePriceInfo.PricePerUnit.USD, 64)
		if err != nil {
			return nil, err
		}
	} else {
		price, err = strconv.ParseFloat(nodePriceInfo.PricePerUnit.CNY, 64)
		if err != nil {
			return nil, err
		}
	}

	return &api.InstancePriceInfo{
		NodeTotalHourlyPrice: float64(price),
		CPUCore:              nodePriceInfo.VCPU,
		CPUCoreHourlyPrice:   float64(price) * e.cpuMemoryCostRatio / (e.cpuMemoryCostRatio + 1),
		RamGiB:               nodePriceInfo.Memory,
		RAMGiBHourlyPrice:    float64(price) / (e.cpuMemoryCostRatio + 1),
		InstanceType:         nodeType,
		BillingMode:          values.BillingModeOnDemand,
		BillingPeriod:        0,
		Region:               e.region,
		CloudProvider:        api.CloudProviderACK,
	}, nil
}
