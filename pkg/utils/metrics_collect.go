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

package utils

import (
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"github.com/kubefin/kubefin/pkg/cloudprice"
	"github.com/kubefin/kubefin/pkg/values"
)

func ParsePodResourceRequest(Containers []v1.Container) (cpu, ram map[string]float64) {
	cpu = make(map[string]float64)
	ram = make(map[string]float64)
	for _, container := range Containers {
		if _, ok := cpu[container.Name]; !ok {
			cpu[container.Name] = 0.0
		}
		if _, ok := ram[container.Name]; !ok {
			ram[container.Name] = 0.0
		}
		cpu[container.Name] += float64(container.Resources.Requests.Cpu().MilliValue()) / values.CoreInMCore
		ram[container.Name] += float64(container.Resources.Requests.Memory().Value()) / values.GBInBytes
	}
	return
}

func ParsePodResourceCost(pod *v1.Pod, provider cloudprice.CloudProviderInterface, lister v12.NodeLister) float64 {
	var cpu, ram float64
	for _, container := range pod.Spec.Containers {
		cpu += float64(container.Resources.Requests.Cpu().MilliValue()) / values.CoreInMCore
		ram += float64(container.Resources.Requests.Memory().Value()) / values.GBInBytes
	}

	if pod.Spec.NodeName == "" {
		return 0
	}

	node, err := lister.Get(pod.Spec.NodeName)
	if err != nil {
		klog.Errorf("failed to get node %s: %v", pod.Spec.NodeName, err)
		return 0
	}
	priceInfo, err := provider.GetNodeHourlyPrice(node)
	if err != nil {
		klog.Errorf("failed to get node %s: %v", pod.Spec.NodeName, err)
		return 0
	}

	cpuCosts := cpu * priceInfo.CPUCoreHourlyPrice
	memoryCosts := ram * priceInfo.RAMGiBHourlyPrice
	return cpuCosts + memoryCosts
}
