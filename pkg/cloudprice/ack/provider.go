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

package ack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"kubefin.dev/kubefin/cmd/kubefin-agent/app/options"
	"kubefin.dev/kubefin/pkg/api"
	cloudpriceapis "kubefin.dev/kubefin/pkg/cloudprice/apis"
	"kubefin.dev/kubefin/pkg/values"
)

const (
	ackClusterIdLabelKey  = "ack.aliyun.com"
	ackNodeTypeLabelKey   = "node.kubernetes.io/instance-type"
	ackNodeRegionLabelKey = "topology.kubernetes.io/region"

	nodePriceQueryUrl = "https://buy-api.aliyun.com/price/getLightWeightPrice2.json?tenant=TenantCalculator"
	nodeSpecQueryUrl  = "https://query.aliyun.com/rest/sell.ecs.allInstanceTypes?domain=aliyun&saleStrategy=PostPaid"
)

type ACKCloudProvider struct {
	client kubernetes.Interface

	cpuMemoryCostRatio float64
	gpuCpuCostRatio    float64
	region             string

	// nodeSpecMap maps [node type]NodeSpec
	nodeSpecMap  map[string]cloudpriceapis.NodeSpec
	nodeSpecLock sync.Mutex
}

func NewACKCloudProvider(client kubernetes.Interface, agentOptions *options.AgentOptions) (*ACKCloudProvider, error) {
	// We have no way to get the cluster name currently
	if agentOptions.ClusterName == "" {
		return nil, fmt.Errorf("please set the cluster name via env CLUSTER_NAME in agent manifest")
	}

	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var clusterId string
	for _, node := range nodes.Items {
		labels := node.Labels
		if len(labels) == 0 {
			continue
		}
		if id, ok := labels[ackClusterIdLabelKey]; ok {
			clusterId = id
			break
		}
	}
	if clusterId == "" {
		return nil, fmt.Errorf("could not find cluster id")
	}
	agentOptions.ClusterId = clusterId

	var region string
	for _, node := range nodes.Items {
		labels := node.Labels
		if len(labels) == 0 {
			continue
		}
		if rg, ok := labels[ackNodeRegionLabelKey]; ok {
			region = rg
			break
		}
	}
	if region == "" {
		return nil, fmt.Errorf("could not find region")
	}

	cpuMemoryCostRatio := cloudpriceapis.DefaultCPUMemoryCostRatio
	if agentOptions.CPUMemoryCostRatio != "" {
		cpuMemoryCostRatio, err = strconv.ParseFloat(agentOptions.CPUMemoryCostRatio, 64)
		if err != nil {
			return nil, err
		}
	}

	gpuCpuCostRatio := cloudpriceapis.DefaultGPUCPUCostRatio
	if agentOptions.GPUCPUCostRatio != "" {
		gpuCpuCostRatio, err = strconv.ParseFloat(agentOptions.GPUCPUCostRatio, 64)
		if err != nil {
			return nil, err
		}
	}

	ackCloud := ACKCloudProvider{
		client:             client,
		region:             region,
		cpuMemoryCostRatio: cpuMemoryCostRatio,
		gpuCpuCostRatio:    gpuCpuCostRatio,
		nodeSpecMap:        map[string]cloudpriceapis.NodeSpec{},
	}
	return &ackCloud, nil
}

func (a *ACKCloudProvider) Start(ctx context.Context) {
}

func (a *ACKCloudProvider) GetNodeHourlyPrice(node *v1.Node) (*api.InstancePriceInfo, error) {
	if node.Labels == nil {
		return nil, fmt.Errorf("node(%s) has no labels", node.Name)
	}

	nodeType, ok := node.Labels[ackNodeTypeLabelKey]
	if !ok {
		return nil, fmt.Errorf("node(%s) has no label %s", node.Name, ackNodeTypeLabelKey)
	}

	a.nodeSpecLock.Lock()
	defer a.nodeSpecLock.Unlock()

	var err error
	nodeSpec, ok := a.nodeSpecMap[nodeType]
	if !ok {
		a.nodeSpecMap, err = queryNodeSpecFromCloud()
		if err != nil {
			klog.Errorf("Query AliCloud ecs spec error:%v", err)
			return nil, err
		}
		if nodeSpec, ok = a.nodeSpecMap[nodeType]; !ok {
			klog.Errorf("Could not find node type:%s", nodeType)
			return nil, fmt.Errorf("could not find node type:%s", nodeType)
		}
	}

	if nodeSpec.Price <= 0 {
		nodePrice, err := queryNodePriceFromCloud(a.region, nodeType)
		if err != nil {
			klog.Errorf("Query AliCloud ecs price error:%v", err)
			return nil, err
		}
		nodeSpec.Price = nodePrice
		a.nodeSpecMap[nodeType] = nodeSpec
	}

	ret := &api.InstancePriceInfo{
		NodeTotalHourlyPrice: nodeSpec.Price,
		CPUCore:              nodeSpec.CPUCount,
		RamGiB:               nodeSpec.RAMGBCount,
		GPUCards:             nodeSpec.GPUAmount,
		InstanceType:         nodeType,
		BillingMode:          values.BillingModeOnDemand,
		BillingPeriod:        0,
		Region:               a.region,
		CloudProvider:        api.CloudProviderACK,
	}

	if ret.GPUCards == 0 {
		ret.CPUCoreHourlyPrice = nodeSpec.Price * a.cpuMemoryCostRatio / (a.cpuMemoryCostRatio + 1)
		ret.RAMGiBHourlyPrice = nodeSpec.Price / (a.cpuMemoryCostRatio + 1)
		ret.GPUCardHourlyPrice = 0
		return ret, nil
	}

	ret.CPUCoreHourlyPrice = nodeSpec.Price * a.cpuMemoryCostRatio / (1 + a.cpuMemoryCostRatio + a.cpuMemoryCostRatio*a.gpuCpuCostRatio)
	ret.RAMGiBHourlyPrice = nodeSpec.Price / (1 + a.cpuMemoryCostRatio + a.cpuMemoryCostRatio*a.gpuCpuCostRatio)
	ret.GPUCardHourlyPrice = nodeSpec.Price * a.cpuMemoryCostRatio * a.gpuCpuCostRatio / (1 + a.cpuMemoryCostRatio + a.cpuMemoryCostRatio*a.gpuCpuCostRatio)
	return ret, nil
}

func queryNodePriceFromCloud(nodeRegion, nodeType string) (float64, error) {
	queryPara := newNodePriceQueryPara(nodeRegion, nodeType)
	jsonData, err := json.Marshal(queryPara)
	if err != nil {
		klog.Errorf("Marshal TenantCalculator error:%v", err)
		return 0, err
	}

	resp, err := http.Post(nodePriceQueryUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		klog.Errorf("Query AliCloud ecs price error:%v", err)
		return 0, err
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Read response body error:%v", err)
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		klog.Errorf("Query AliCloud ecs price error:%s", string(data))
		return 0, fmt.Errorf("query AliCloud ecs price error:%s", string(data))
	}

	priceResult := &TenantCalculatorResult{}
	err = json.Unmarshal(data, priceResult)
	if err != nil {
		klog.Errorf("Unmarshal response body error:%v", err)
		return 0, err
	}

	return priceResult.Data.Order.TradeAmount, nil
}

func queryNodeSpecFromCloud() (map[string]cloudpriceapis.NodeSpec, error) {
	resp, err := http.Get(nodeSpecQueryUrl)
	if err != nil {
		klog.Errorf("Query AliCloud ecs spec error:%v", err)
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Read response body error:%v", err)
		return nil, err
	}

	queryResult := &NodeSpecQueryResult{}
	err = json.Unmarshal(data, queryResult)
	if err != nil {
		klog.Errorf("Unmarshal response body error:%v", err)
		return nil, err
	}

	ret := map[string]cloudpriceapis.NodeSpec{}
	for _, instance := range queryResult.Data.Components.InstanceType.InstanceType {
		nodeType := instance.InstanceTypeId
		if _, ok := ret[nodeType]; !ok {
			cpuCount, err := strconv.ParseFloat(instance.CPUCoreCount, 64)
			if err != nil {
				klog.Errorf("Can not parse cpu count(%s):%v", instance.CPUCoreCount, err)
				return nil, err
			}
			memoryCount, err := strconv.ParseFloat(instance.MemorySize, 64)
			if err != nil {
				klog.Errorf("Can not parse memory count(%s):%v", instance.MemorySize, err)
				return nil, err
			}
			gpuAmount, err := strconv.ParseFloat(instance.GPUAmount, 64)
			if err != nil {
				klog.Errorf("Can not parse gpu amount(%s):%v", instance.GPUAmount, err)
				return nil, err
			}
			ret[nodeType] = cloudpriceapis.NodeSpec{
				CPUCount:   cpuCount,
				RAMGBCount: memoryCount,
				GPUAmount:  gpuAmount,
			}
		}
	}
	return ret, nil
}

func newNodePriceQueryPara(instanceRegion, instanceType string) *TenantCalculator {
	calculator := &TenantCalculator{
		Tenant: "TenantCalculator",
		Configurations: []TenantCalculatorConfiguration{
			{
				CommodityCode:   "ecs",
				SpecCode:        "ecs",
				ChargeType:      "POSTPAY",
				OrderType:       "BUY",
				Quantity:        1,
				Duration:        1,
				PricingCycle:    "Hour",
				UseTimeUnit:     "Hour",
				UseTimeQuantity: 1,
				Components: []TenantCalculatorComponent{
					{
						ComponentCode: "vm_region_no",
						InstanceProperty: []TenantCalculatorInstanceProperty{
							{Code: "vm_region_no", Value: instanceRegion},
						},
					},
					{
						ComponentCode: "instance_type",
						InstanceProperty: []TenantCalculatorInstanceProperty{
							{Code: "instance_type", Value: instanceType},
						},
					},
				},
			},
		},
	}

	return calculator
}
