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

package core

import (
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubefin/kubefin/cmd/kubefin-agent/app/options"
	"github.com/kubefin/kubefin/pkg/api"
	"github.com/kubefin/kubefin/pkg/cloudprice"
	metricscache "github.com/kubefin/kubefin/pkg/metrics/cache"
	"github.com/kubefin/kubefin/pkg/utils"
	"github.com/kubefin/kubefin/pkg/values"
)

var (
	metricsCostLabelKey = []string{
		values.NodeNameLabelKey,
		values.NodeInstanceTypeLabelKey,
		values.BillingModeLabelKey,
		values.NodeBillingPeriodLabelKey,
		values.RegionLabelKey,
		values.CloudProviderLabelKey,
		values.ClusterNameLabelKey,
		values.ClusterIdLabelKey,
	}
	metricsCostUnifiedLabelKey = []string{
		values.NodeNameLabelKey,
		values.NodeInstanceTypeLabelKey,
		values.BillingModeLabelKey,
		values.NodeBillingPeriodLabelKey,
		values.RegionLabelKey,
		values.CloudProviderLabelKey,
		values.ClusterNameLabelKey,
		values.ClusterIdLabelKey,
		values.ResourceTypeLabelKey,
	}
	resourceMetricsLabelKey = []string{
		values.NodeNameLabelKey,
		values.ClusterNameLabelKey,
		values.ClusterIdLabelKey,
		values.ResourceTypeLabelKey,
		values.BillingModeLabelKey,
	}
	nodeCPUCoreHourlyCostDesc = prometheus.NewDesc(
		values.NodeCPUCoreHourlyCostMetricsName,
		"The node hourly cpu-core cost for the node", metricsCostLabelKey, nil)
	nodeRAMGBHourlyCostDesc = prometheus.NewDesc(
		values.NodeRAMGBHourlyCostMetricsName,
		"The node hourly ram-gb cost for the node", metricsCostLabelKey, nil)
	nodeTotalCostDesc = prometheus.NewDesc(
		values.NodeTotalHourlyCostMetricsName,
		"The node total hourly cost for the node", metricsCostLabelKey, nil)
	nodeResourceHourlyCostDesc = prometheus.NewDesc(
		values.NodeResourceHourlyCostMetricsName,
		"The node hourly cpu/ram(total cores) cost for the node", metricsCostUnifiedLabelKey, nil)
	nodeResourceTotalDesc = prometheus.NewDesc(
		values.NodeResourceTotalMetricsName,
		"The total node resource for the node", resourceMetricsLabelKey, nil)
	nodeResourceSystemTakenDesc = prometheus.NewDesc(
		values.NodeResourceSystemTakenName,
		"The total node resoruce taken by system", resourceMetricsLabelKey, nil)
	nodeResourceAvailableDesc = prometheus.NewDesc(
		values.NodeResourceAvailableMetricsName,
		"The node resource allocatable for the node", resourceMetricsLabelKey, nil)
	nodeResourceUsageDesc = prometheus.NewDesc(
		values.NodeResourceUsageMetricsName,
		"The node resource usage for the node", resourceMetricsLabelKey, nil)
	nodeResourceRequestedDesc = prometheus.NewDesc(
		values.NodeResourceRequestedName,
		"The node resoruce requested for the node", resourceMetricsLabelKey, nil)
)

type nodeResourceInfo struct {
	allocatableResource corev1.ResourceList
	requestedResource   corev1.ResourceList
}

type nodeMetricsCollector struct {
	clusterName string
	clusterId   string

	usageMetricsCache *metricscache.ClusterResourceUsageMetricsCache
	provider          cloudprice.CloudProviderInterface
	nodeLister        v1.NodeLister

	mutex        sync.Mutex
	nodeResouece map[string]nodeResourceInfo
}

func (n *nodeMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- nodeCPUCoreHourlyCostDesc
	ch <- nodeRAMGBHourlyCostDesc
	ch <- nodeTotalCostDesc
	ch <- nodeResourceHourlyCostDesc
	ch <- nodeResourceTotalDesc
	ch <- nodeResourceSystemTakenDesc
	ch <- nodeResourceAvailableDesc
	ch <- nodeResourceUsageDesc
	ch <- nodeResourceRequestedDesc
}

func (n *nodeMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	n.collectNodeCost(ch)
	n.collectNodeResourceUsage(ch)
	n.collectNodeResourceMetrics(ch)
}

func (n *nodeMetricsCollector) handleNodeAddition(node *corev1.Node) {
	if _, ok := n.nodeResouece[node.Name]; !ok {
		n.mutex.Lock()
		defer n.mutex.Unlock()
		n.nodeResouece[node.Name] = nodeResourceInfo{
			allocatableResource: corev1.ResourceList{},
			requestedResource:   corev1.ResourceList{},
		}

		for resourceName, resourceValue := range node.Status.Allocatable {
			n.nodeResouece[node.Name].allocatableResource[resourceName] = resourceValue
		}
	}
}

func (n *nodeMetricsCollector) handleNodeDeletion(node *corev1.Node) {
	if _, ok := n.nodeResouece[node.Name]; !ok {
		n.mutex.Lock()
		defer n.mutex.Unlock()
		delete(n.nodeResouece, node.Name)
	}
}

func (n *nodeMetricsCollector) addPodResourceRequested(pod *corev1.Pod) {
	for _, container := range pod.Spec.Containers {
		for resourceName, resourceValue := range container.Resources.Requests {
			requested, ok := n.nodeResouece[pod.Spec.NodeName].requestedResource[resourceName]
			if !ok {
				requested = resource.Quantity{}
			}
			requested.Add(resourceValue)
			n.nodeResouece[pod.Spec.NodeName].requestedResource[resourceName] = requested
		}
	}
}

func (n *nodeMetricsCollector) handlePodAddition(pod *corev1.Pod) {
	if pod.Spec.NodeName != "" {
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return
		}
		n.mutex.Lock()
		defer n.mutex.Unlock()

		if _, ok := n.nodeResouece[pod.Spec.NodeName]; !ok {
			klog.Warningf("Node %s not found in cluster", pod.Spec.NodeName)
		}
		n.addPodResourceRequested(pod)
	}
}

func (n *nodeMetricsCollector) handlePodUpdate(oldPod, newPod *corev1.Pod) {
	if newPod.Spec.NodeName == "" {
		return
	}

	if oldPod.Spec.NodeName == "" {
		if newPod.Status.Phase == corev1.PodFailed || newPod.Status.Phase == corev1.PodSucceeded {
			return
		}
		n.mutex.Lock()
		defer n.mutex.Unlock()

		if _, ok := n.nodeResouece[newPod.Spec.NodeName]; !ok {
			klog.Warningf("Node %s not found in cluster", newPod.Spec.NodeName)
		}
		n.addPodResourceRequested(newPod)
		return
	}

	// Referring issue: https://github.com/kubefin/kubefin/issues/28
	if (oldPod.Status.Phase != corev1.PodFailed && oldPod.Status.Phase != corev1.PodSucceeded) &&
		(newPod.Status.Phase == corev1.PodFailed || newPod.Status.Phase == corev1.PodSucceeded) {
		n.deletePodResourceRequested(newPod)
	}
}

func (n *nodeMetricsCollector) deletePodResourceRequested(pod *corev1.Pod) {
	for _, container := range pod.Spec.Containers {
		for resourceName, resourceValue := range container.Resources.Requests {
			requested, ok := n.nodeResouece[pod.Spec.NodeName].requestedResource[resourceName]
			if !ok {
				continue
			}
			requested.Sub(resourceValue)
			n.nodeResouece[pod.Spec.NodeName].requestedResource[resourceName] = requested
		}
	}
}

func (n *nodeMetricsCollector) handlePodDeletion(pod *corev1.Pod) {
	if pod.Spec.NodeName != "" {
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return
		}
		n.mutex.Lock()
		defer n.mutex.Unlock()

		n.deletePodResourceRequested(pod)
	}
}

func (n *nodeMetricsCollector) collectNodeCost(ch chan<- prometheus.Metric) {
	nodes, err := n.nodeLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("List all nodes error:%v", err)
		return
	}

	for _, node := range nodes {
		nodeCostInfo, err := n.provider.GetNodeHourlyPrice(node)
		if err != nil {
			klog.Errorf("Get node price from cloud provider error:%v", err)
			continue
		}
		metricsLabelValues := prometheus.Labels{
			values.NodeNameLabelKey:          node.Name,
			values.NodeInstanceTypeLabelKey:  nodeCostInfo.InstanceType,
			values.BillingModeLabelKey:       nodeCostInfo.BillingMode,
			values.NodeBillingPeriodLabelKey: strconv.Itoa(nodeCostInfo.BillingPeriod),
			values.RegionLabelKey:            nodeCostInfo.Region,
			values.CloudProviderLabelKey:     nodeCostInfo.CloudProvider,
			values.ClusterNameLabelKey:       n.clusterName,
			values.ClusterIdLabelKey:         n.clusterId,
		}
		ch <- prometheus.MustNewConstMetric(nodeCPUCoreHourlyCostDesc,
			prometheus.GaugeValue, nodeCostInfo.CPUCoreHourlyPrice, utils.ConvertPrometheusLabelValuesInOrder(metricsCostLabelKey, metricsLabelValues)...)
		ch <- prometheus.MustNewConstMetric(nodeRAMGBHourlyCostDesc,
			prometheus.GaugeValue, nodeCostInfo.RAMGiBHourlyPrice, utils.ConvertPrometheusLabelValuesInOrder(metricsCostLabelKey, metricsLabelValues)...)
		ch <- prometheus.MustNewConstMetric(nodeTotalCostDesc,
			prometheus.GaugeValue, nodeCostInfo.NodeTotalHourlyPrice, utils.ConvertPrometheusLabelValuesInOrder(metricsCostLabelKey, metricsLabelValues)...)

		metricsLabelValues[values.ResourceTypeLabelKey] = string(corev1.ResourceCPU)
		ch <- prometheus.MustNewConstMetric(nodeResourceHourlyCostDesc,
			prometheus.GaugeValue, nodeCostInfo.CPUCoreHourlyPrice*nodeCostInfo.CPUCore, utils.ConvertPrometheusLabelValuesInOrder(metricsCostUnifiedLabelKey, metricsLabelValues)...)
		metricsLabelValues[values.ResourceTypeLabelKey] = string(corev1.ResourceMemory)
		ch <- prometheus.MustNewConstMetric(nodeResourceHourlyCostDesc,
			prometheus.GaugeValue, nodeCostInfo.RAMGiBHourlyPrice*nodeCostInfo.RamGiB, utils.ConvertPrometheusLabelValuesInOrder(metricsCostUnifiedLabelKey, metricsLabelValues)...)
	}
}

func (n *nodeMetricsCollector) collectNodeResourceUsage(ch chan<- prometheus.Metric) {
	nodes := n.usageMetricsCache.QueryAllNodesUsage()
	for _, node := range nodes {
		nodeCostInfo, err := n.getNodeCostInfo(node.ResourceName)
		if err != nil {
			continue
		}

		metricsLabels := prometheus.Labels{
			values.NodeNameLabelKey:    node.ResourceName,
			values.ClusterNameLabelKey: n.clusterName,
			values.ClusterIdLabelKey:   n.clusterId,
			values.BillingModeLabelKey: nodeCostInfo.BillingMode,
		}
		metricsLabels[values.ResourceTypeLabelKey] = string(corev1.ResourceCPU)
		ch <- prometheus.MustNewConstMetric(nodeResourceUsageDesc,
			prometheus.GaugeValue, node.CPUUsage, utils.ConvertPrometheusLabelValuesInOrder(resourceMetricsLabelKey, metricsLabels)...)
		metricsLabels[values.ResourceTypeLabelKey] = string(corev1.ResourceMemory)
		ch <- prometheus.MustNewConstMetric(nodeResourceUsageDesc,
			prometheus.GaugeValue, node.MemoryUsage, utils.ConvertPrometheusLabelValuesInOrder(resourceMetricsLabelKey, metricsLabels)...)
	}
}

func (n *nodeMetricsCollector) collectNodeResourceMetrics(ch chan<- prometheus.Metric) {
	nodes, err := n.nodeLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("List all nodes error:%v", err)
		return
	}

	for _, node := range nodes {
		nodeCostInfo, err := n.getNodeCostInfo(node.Name)
		if err != nil {
			continue
		}

		metricsLabels := prometheus.Labels{
			values.NodeNameLabelKey:    node.Name,
			values.ClusterNameLabelKey: n.clusterName,
			values.ClusterIdLabelKey:   n.clusterId,
			values.BillingModeLabelKey: nodeCostInfo.BillingMode,
		}

		metricsLabels[values.ResourceTypeLabelKey] = string(corev1.ResourceCPU)
		ch <- prometheus.MustNewConstMetric(nodeResourceTotalDesc,
			prometheus.GaugeValue, nodeCostInfo.CPUCore, utils.ConvertPrometheusLabelValuesInOrder(resourceMetricsLabelKey, metricsLabels)...)

		n.mutex.Lock()
		if _, ok := n.nodeResouece[node.Name]; ok {
			allocatable := n.nodeResouece[node.Name].allocatableResource[corev1.ResourceCPU]
			requested := n.nodeResouece[node.Name].requestedResource[corev1.ResourceCPU]

			resourceSystemTaken := nodeCostInfo.CPUCore - utils.ConvertQualityToCore(&allocatable)
			allocatable.Sub(requested)

			ch <- prometheus.MustNewConstMetric(nodeResourceRequestedDesc,
				prometheus.GaugeValue, utils.ConvertQualityToCore(&requested), utils.ConvertPrometheusLabelValuesInOrder(resourceMetricsLabelKey, metricsLabels)...)
			ch <- prometheus.MustNewConstMetric(nodeResourceSystemTakenDesc,
				prometheus.GaugeValue, resourceSystemTaken, utils.ConvertPrometheusLabelValuesInOrder(resourceMetricsLabelKey, metricsLabels)...)
			ch <- prometheus.MustNewConstMetric(nodeResourceAvailableDesc,
				prometheus.GaugeValue, utils.ConvertQualityToCore(&allocatable), utils.ConvertPrometheusLabelValuesInOrder(resourceMetricsLabelKey, metricsLabels)...)
		}
		n.mutex.Unlock()

		metricsLabels[values.ResourceTypeLabelKey] = string(corev1.ResourceMemory)
		ch <- prometheus.MustNewConstMetric(nodeResourceTotalDesc,
			prometheus.GaugeValue, nodeCostInfo.RamGiB, utils.ConvertPrometheusLabelValuesInOrder(resourceMetricsLabelKey, metricsLabels)...)

		n.mutex.Lock()
		if _, ok := n.nodeResouece[node.Name]; ok {
			allocatable := n.nodeResouece[node.Name].allocatableResource[corev1.ResourceMemory]
			requested := n.nodeResouece[node.Name].requestedResource[corev1.ResourceMemory]

			resourceSystemTaken := nodeCostInfo.RamGiB - utils.ConvertQualityToGiB(&allocatable)
			allocatable.Sub(requested)

			ch <- prometheus.MustNewConstMetric(nodeResourceRequestedDesc,
				prometheus.GaugeValue, utils.ConvertQualityToGiB(&requested), utils.ConvertPrometheusLabelValuesInOrder(resourceMetricsLabelKey, metricsLabels)...)
			ch <- prometheus.MustNewConstMetric(nodeResourceSystemTakenDesc,
				prometheus.GaugeValue, resourceSystemTaken, utils.ConvertPrometheusLabelValuesInOrder(resourceMetricsLabelKey, metricsLabels)...)
			ch <- prometheus.MustNewConstMetric(nodeResourceAvailableDesc,
				prometheus.GaugeValue, utils.ConvertQualityToGiB(&allocatable), utils.ConvertPrometheusLabelValuesInOrder(resourceMetricsLabelKey, metricsLabels)...)
		}
		n.mutex.Unlock()
	}
}

func (n *nodeMetricsCollector) getNodeCostInfo(nodeName string) (*api.InstancePriceInfo, error) {
	node, err := n.nodeLister.Get(nodeName)
	if err != nil {
		klog.Errorf("Get node from lister error:%v", err)
		return nil, err
	}

	nodeCostInfo, err := n.provider.GetNodeHourlyPrice(node)
	if err != nil {
		klog.Errorf("Get node price from cloud provider error:%v", err)
		return nil, err
	}
	return nodeCostInfo, nil
}

func RegisterNodeLevelMetricsCollection(agentOptions *options.AgentOptions,
	provider cloudprice.CloudProviderInterface,
	coreResourceInformerLister *api.CoreResourceInformerLister,
	usageMetricsCache *metricscache.ClusterResourceUsageMetricsCache) {
	nodeMetricsCollector := &nodeMetricsCollector{
		clusterName:       agentOptions.ClusterName,
		clusterId:         agentOptions.ClusterId,
		usageMetricsCache: usageMetricsCache,
		provider:          provider,
		nodeLister:        coreResourceInformerLister.NodeLister,
		nodeResouece:      make(map[string]nodeResourceInfo),
	}

	nodeMetricsCollector.registerNodeResourceEventHandler(coreResourceInformerLister)
	prometheus.MustRegister(nodeMetricsCollector)
}

func (n *nodeMetricsCollector) registerNodeResourceEventHandler(coreResourceInformerLister *api.CoreResourceInformerLister) {
	coreResourceInformerLister.NodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node, ok := obj.(*corev1.Node)
			if !ok {
				return
			}
			n.handleNodeAddition(node)
		},
		DeleteFunc: func(obj interface{}) {
			node, ok := obj.(*corev1.Node)
			if !ok {
				return
			}
			n.handleNodeDeletion(node)
		},
	})

	coreResourceInformerLister.PodInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				return
			}
			n.handlePodAddition(pod)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newPod, ok := newObj.(*corev1.Pod)
			if !ok {
				return
			}
			oldPod, ok := oldObj.(*corev1.Pod)
			if !ok {
				return
			}
			n.handlePodUpdate(oldPod, newPod)
		},
		DeleteFunc: func(obj interface{}) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				return
			}
			n.handlePodDeletion(pod)
		},
	})
}
