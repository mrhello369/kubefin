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
	"encoding/json"

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"github.com/kubefin/kubefin/cmd/kubefin-agent/app/options"
	"github.com/kubefin/kubefin/pkg/api"
	"github.com/kubefin/kubefin/pkg/cloudprice"
	metricscache "github.com/kubefin/kubefin/pkg/metrics/cache"
	"github.com/kubefin/kubefin/pkg/utils"
	"github.com/kubefin/kubefin/pkg/values"
)

var (
	containerNoneCareLabelKey = []string{
		values.NamespaceLabelKey,
		values.PodNameLabelKey,
		values.ClusterNameLabelKey,
		values.ClusterIdLabelKey,
		values.ResourceTypeLabelKey,
		values.PodScheduledKey,
		values.LabelsLabelKey,
	}
	containerCareLabelKey = []string{
		values.NamespaceLabelKey,
		values.PodNameLabelKey,
		values.ClusterNameLabelKey,
		values.ClusterIdLabelKey,
		values.ResourceTypeLabelKey,
		values.LabelsLabelKey,
		// For multiple container pod, this metrics is needed for cpu/memory size recommendation
		values.ContainerNameLabelKey,
	}

	podResourceCostDesc = prometheus.NewDesc(
		values.PodResoueceCostMetricsName,
		"The pod level resource cost",
		containerNoneCareLabelKey, nil)
	podResourceRequestDesc = prometheus.NewDesc(
		values.PodResourceRequestMetricsName,
		"The pod container level resource requested",
		containerCareLabelKey, nil)
	podResourceUsageDesc = prometheus.NewDesc(
		values.PodResourceUsageMetricsName,
		"The pod container level resource usage",
		containerCareLabelKey, nil)
)

type podMetricsCollector struct {
	clusterId   string
	clusterName string

	usageMetricsCache *metricscache.ClusterResourceUsageMetricsCache
	provider          cloudprice.CloudProviderInterface

	podLister  v1.PodLister
	nodeLister v1.NodeLister
}

func (p *podMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- podResourceCostDesc
	ch <- podResourceRequestDesc
	ch <- podResourceUsageDesc
}

func (p *podMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	pods, err := p.podLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("List all pods error:%v", err)
		return
	}
	for _, pod := range pods {
		podLabels, err := json.Marshal(pod.Labels)
		if err != nil {
			klog.Errorf("Marshal pod labels error:%v", err)
			return
		}
		cost := 0.0
		scheduled := "false"
		if pod.Spec.NodeName != "" {
			cost = utils.ParsePodResourceCost(pod, p.provider, p.nodeLister)
			scheduled = "true"
		}
		crCareLabels := prometheus.Labels{
			values.NamespaceLabelKey:    pod.Namespace,
			values.PodNameLabelKey:      pod.Name,
			values.ClusterNameLabelKey:  p.clusterName,
			values.ClusterIdLabelKey:    p.clusterId,
			values.LabelsLabelKey:       string(podLabels),
			values.PodScheduledKey:      scheduled,
			values.ResourceTypeLabelKey: "cost",
		}
		ch <- prometheus.MustNewConstMetric(podResourceCostDesc,
			prometheus.GaugeValue, cost, utils.ConvertPrometheusLabelValuesInOrder(containerNoneCareLabelKey, crCareLabels)...)

		crNoneCareLabels := prometheus.Labels{
			values.NamespaceLabelKey:   pod.Namespace,
			values.PodNameLabelKey:     pod.Name,
			values.ClusterNameLabelKey: p.clusterName,
			values.ClusterIdLabelKey:   p.clusterId,
			values.LabelsLabelKey:      string(podLabels),
		}
		cpuRequest, memoryRequest := utils.ParsePodResourceRequest(pod.Spec.Containers)

		crNoneCareLabels[values.ResourceTypeLabelKey] = string(corev1.ResourceCPU)
		for containerName, cpu := range cpuRequest {
			crNoneCareLabels[values.ContainerNameLabelKey] = containerName
			ch <- prometheus.MustNewConstMetric(podResourceRequestDesc,
				prometheus.GaugeValue, cpu, utils.ConvertPrometheusLabelValuesInOrder(containerCareLabelKey, crNoneCareLabels)...)
		}
		crNoneCareLabels[values.ResourceTypeLabelKey] = string(corev1.ResourceMemory)
		for containerName, memory := range memoryRequest {
			crNoneCareLabels[values.ContainerNameLabelKey] = containerName
			ch <- prometheus.MustNewConstMetric(podResourceRequestDesc,
				prometheus.GaugeValue, memory, utils.ConvertPrometheusLabelValuesInOrder(containerCareLabelKey, crNoneCareLabels)...)
		}
	}
	p.CollectPodResourceUsage(ch)
}

func (p *podMetricsCollector) CollectPodResourceUsage(ch chan<- prometheus.Metric) {
	pods := p.usageMetricsCache.QueryAllPodsUsage()
	for _, pod := range pods {
		podStandard, err := p.podLister.Pods(pod.Namespace).Get(pod.Name)
		if err != nil {
			klog.Errorf("Get pod error:%v", err)
			return
		}
		podLabels, err := json.Marshal(podStandard.Labels)
		if err != nil {
			klog.Errorf("Marshal pod labels error:%v", err)
			return
		}
		labels := prometheus.Labels{
			values.NamespaceLabelKey:   pod.Namespace,
			values.PodNameLabelKey:     pod.Name,
			values.ClusterNameLabelKey: p.clusterName,
			values.ClusterIdLabelKey:   p.clusterId,
			values.LabelsLabelKey:      string(podLabels),
		}

		labels[values.ResourceTypeLabelKey] = string(corev1.ResourceCPU)
		for _, ctrInfo := range pod.ContainersUsage {
			labels[values.ContainerNameLabelKey] = ctrInfo.ResourceName
			ch <- prometheus.MustNewConstMetric(podResourceUsageDesc,
				prometheus.GaugeValue, ctrInfo.CPUUsage, utils.ConvertPrometheusLabelValuesInOrder(containerCareLabelKey, labels)...)
		}
		labels[values.ResourceTypeLabelKey] = string(corev1.ResourceMemory)
		for _, ctrInfo := range pod.ContainersUsage {
			labels[values.ContainerNameLabelKey] = ctrInfo.ResourceName
			ch <- prometheus.MustNewConstMetric(podResourceUsageDesc,
				prometheus.GaugeValue, ctrInfo.MemoryUsage, utils.ConvertPrometheusLabelValuesInOrder(containerCareLabelKey, labels)...)
		}
	}
}

func RegisterPodLevelMetricsCollection(agentOptions *options.AgentOptions,
	provider cloudprice.CloudProviderInterface,
	coreResourceInformerLister *api.CoreResourceInformerLister,
	usageMetricsCache *metricscache.ClusterResourceUsageMetricsCache) {
	podMetricsCollector := &podMetricsCollector{
		clusterId:         agentOptions.ClusterId,
		clusterName:       agentOptions.ClusterName,
		usageMetricsCache: usageMetricsCache,
		provider:          provider,
		podLister:         coreResourceInformerLister.PodLister,
		nodeLister:        coreResourceInformerLister.NodeLister,
	}

	prometheus.MustRegister(podMetricsCollector)
}
