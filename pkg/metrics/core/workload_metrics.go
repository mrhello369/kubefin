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
	"context"
	"encoding/json"

	"github.com/prometheus/client_golang/prometheus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersappv1 "k8s.io/client-go/listers/apps/v1"
	listercorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/kubefin/kubefin/cmd/kubefin-agent/app/options"
	"github.com/kubefin/kubefin/pkg/api"
	"github.com/kubefin/kubefin/pkg/cloudprice"
	"github.com/kubefin/kubefin/pkg/utils"
	"github.com/kubefin/kubefin/pkg/values"
)

var (
	workloadCRNoneCareLabelKey = []string{
		values.WorkloadTypeLabelKey,
		values.WorkloadNameLabelKey,
		values.NamespaceLabelKey,
		values.ClusterNameLabelKey,
		values.ClusterIdLabelKey,
		values.LabelsLabelKey,
		values.ResourceTypeLabelKey,
	}
	workloadResourceCostDesc = prometheus.NewDesc(
		values.WorkloadResourceCostMetricsName,
		"The workload resource cost", workloadCRNoneCareLabelKey, nil)
	workloadPodCountDesc = prometheus.NewDesc(
		values.WorkloadPodCountMetricsName,
		"The workload pod count", workloadCRNoneCareLabelKey, nil)

	workloadCRCareLabelKey = []string{
		values.WorkloadTypeLabelKey,
		values.WorkloadNameLabelKey,
		values.NamespaceLabelKey,
		values.ClusterNameLabelKey,
		values.ClusterIdLabelKey,
		values.LabelsLabelKey,
		values.ResourceTypeLabelKey,
		// For multiple container workload, this metrics is needed for cpu/memory size recommendation
		values.ContainerNameLabelKey,
	}
	workloadResourceRequestDesc = prometheus.NewDesc(
		values.WorkloadResourceRequestMetricsName,
		"The workload resource request", workloadCRCareLabelKey, nil)
	workloadResourceUsageDesc = prometheus.NewDesc(
		values.WorkloadResourceUsageMetricsName,
		"The workload resource usage", workloadCRCareLabelKey, nil)
)

type workloadMetricsCollector struct {
	clusterName string
	clusterId   string

	metricsClient *versioned.Clientset
	provider      cloudprice.CloudProviderInterface

	podLister         listercorev1.PodLister
	nodeLister        listercorev1.NodeLister
	daemonSetLister   listersappv1.DaemonSetLister
	statefulSetLister listersappv1.StatefulSetLister
	deploymentLister  listersappv1.DeploymentLister
}

func (w *workloadMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- workloadResourceCostDesc
	ch <- workloadPodCountDesc
	ch <- workloadResourceRequestDesc
	ch <- workloadResourceUsageDesc
}

func (w *workloadMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	w.collectDeploymentResourceMetrics(ch)
	w.collectDaemonSetResourceMetrics(ch)
	w.collectStatefulSetResourceMetrics(ch)
}

func (w *workloadMetricsCollector) collectDaemonSetResourceMetrics(ch chan<- prometheus.Metric) {
	daemonSets, err := w.daemonSetLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("List all daemonSets error:%v", err)
		return
	}
	for _, daemonSet := range daemonSets {
		daemonSetLabels, err := json.Marshal(daemonSet.Labels)
		if err != nil {
			klog.Errorf("Marshal daemonSet labels error:%v", err)
			continue
		}
		selector, err := metav1.LabelSelectorAsSelector(daemonSet.Spec.Selector)
		if err != nil {
			klog.Errorf("Parse deployment selector error:%v", err)
			return
		}

		containerNoneCareLabelValues := prometheus.Labels{
			values.WorkloadTypeLabelKey: "daemonset",
			values.WorkloadNameLabelKey: daemonSet.Name,
			values.NamespaceLabelKey:    daemonSet.Namespace,
			values.ClusterNameLabelKey:  w.clusterName,
			values.ClusterIdLabelKey:    w.clusterId,
			values.LabelsLabelKey:       string(daemonSetLabels),
		}
		w.collectDaemonSetPodMetrics(daemonSet, selector, containerNoneCareLabelValues, ch)
		w.collectDaemonSetCostMetrics(daemonSet, selector, containerNoneCareLabelValues, ch)

		containerCareLabelValues := prometheus.Labels{
			values.WorkloadTypeLabelKey: "daemonset",
			values.WorkloadNameLabelKey: daemonSet.Name,
			values.NamespaceLabelKey:    daemonSet.Namespace,
			values.ClusterNameLabelKey:  w.clusterName,
			values.ClusterIdLabelKey:    w.clusterId,
			values.LabelsLabelKey:       string(daemonSetLabels),
		}
		w.collectDaemonSetRequestMetrics(daemonSet, selector, containerCareLabelValues, ch)
		w.collectDaemonSetUsageMetrics(daemonSet, selector, containerCareLabelValues, ch)
	}
}

func (w *workloadMetricsCollector) collectDaemonSetPodMetrics(ds *appsv1.DaemonSet, selector labels.Selector,
	labels prometheus.Labels, ch chan<- prometheus.Metric) {
	pods, err := w.podLister.Pods(ds.Namespace).List(selector)
	if err != nil {
		klog.Errorf("List all pods error:%v", err)
		return
	}
	podCount := float64(len(pods))
	labels[values.ResourceTypeLabelKey] = "pod"
	ch <- prometheus.MustNewConstMetric(workloadPodCountDesc,
		prometheus.GaugeValue, podCount, utils.ConvertPrometheusLabelValuesInOrder(workloadCRNoneCareLabelKey, labels)...)
}

func (w *workloadMetricsCollector) collectDaemonSetRequestMetrics(ds *appsv1.DaemonSet, selector labels.Selector,
	labels prometheus.Labels, ch chan<- prometheus.Metric) {
	pods, err := w.podLister.Pods(ds.Namespace).List(selector)
	if err != nil {
		klog.Errorf("List all pods error:%v", err)
		return
	}

	cpuTotalRequest, ramTotalRequest := map[string]float64{}, map[string]float64{}
	for _, pod := range pods {
		cpu, ram := utils.ParsePodResourceRequest(pod.Spec.Containers)
		for containerName, value := range cpu {
			if _, ok := cpuTotalRequest[containerName]; !ok {
				cpuTotalRequest[containerName] = 0
			}
			cpuTotalRequest[containerName] += value
		}
		for containerName, value := range ram {
			if _, ok := ramTotalRequest[containerName]; !ok {
				ramTotalRequest[containerName] = 0
			}
			ramTotalRequest[containerName] += value
		}
	}

	labels[values.ResourceTypeLabelKey] = string(corev1.ResourceCPU)
	for containerName, cpu := range cpuTotalRequest {
		labels[values.ContainerNameLabelKey] = containerName
		ch <- prometheus.MustNewConstMetric(workloadResourceRequestDesc,
			prometheus.GaugeValue, cpu, utils.ConvertPrometheusLabelValuesInOrder(workloadCRCareLabelKey, labels)...)
	}
	labels[values.ResourceTypeLabelKey] = string(corev1.ResourceMemory)
	for containerName, ram := range ramTotalRequest {
		labels[values.ContainerNameLabelKey] = containerName
		ch <- prometheus.MustNewConstMetric(workloadResourceRequestDesc,
			prometheus.GaugeValue, ram, utils.ConvertPrometheusLabelValuesInOrder(workloadCRCareLabelKey, labels)...)
	}
}

func (w *workloadMetricsCollector) collectDaemonSetUsageMetrics(ds *appsv1.DaemonSet, selector labels.Selector,
	labels prometheus.Labels, ch chan<- prometheus.Metric) {
	pods, err := w.metricsClient.MetricsV1beta1().PodMetricses(ds.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		klog.Errorf("List all pod metrics error:%v, kubernetes metrics server may not be installed", err)
		return
	}
	cpuTotalUsage, memoryTotalUsage := map[string]float64{}, map[string]float64{}
	for _, pod := range pods.Items {
		cpu, ram := utils.ParsePodResourceUsage(pod.Containers)
		for containerName, value := range cpu {
			if _, ok := cpuTotalUsage[containerName]; !ok {
				cpuTotalUsage[containerName] = 0
			}
			cpuTotalUsage[containerName] += value
		}
		for containerName, value := range ram {
			if _, ok := memoryTotalUsage[containerName]; !ok {
				memoryTotalUsage[containerName] = 0
			}
			memoryTotalUsage[containerName] += value
		}
	}

	labels[values.ResourceTypeLabelKey] = string(corev1.ResourceCPU)
	for containerName, cpu := range cpuTotalUsage {
		labels[values.ContainerNameLabelKey] = containerName
		ch <- prometheus.MustNewConstMetric(workloadResourceUsageDesc,
			prometheus.GaugeValue, cpu, utils.ConvertPrometheusLabelValuesInOrder(workloadCRCareLabelKey, labels)...)
	}

	labels[values.ResourceTypeLabelKey] = string(corev1.ResourceMemory)
	for containerName, ram := range memoryTotalUsage {
		labels[values.ContainerNameLabelKey] = containerName
		ch <- prometheus.MustNewConstMetric(workloadResourceUsageDesc,
			prometheus.GaugeValue, ram, utils.ConvertPrometheusLabelValuesInOrder(workloadCRCareLabelKey, labels)...)
	}
}

func (w *workloadMetricsCollector) collectDaemonSetCostMetrics(ds *appsv1.DaemonSet, selector labels.Selector,
	labels prometheus.Labels, ch chan<- prometheus.Metric) {
	pods, err := w.podLister.Pods(ds.Namespace).List(selector)
	if err != nil {
		klog.Errorf("List all pods error:%v", err)
		return
	}

	totalCost := 0.0
	for _, pod := range pods {
		if pod.Spec.NodeName == "" {
			continue
		}
		totalCost += utils.ParsePodResourceCost(pod, w.provider, w.nodeLister)
	}

	labels[values.ResourceTypeLabelKey] = "cost"
	ch <- prometheus.MustNewConstMetric(workloadResourceCostDesc,
		prometheus.GaugeValue, totalCost, utils.ConvertPrometheusLabelValuesInOrder(workloadCRNoneCareLabelKey, labels)...)
}

func (w *workloadMetricsCollector) collectStatefulSetResourceMetrics(ch chan<- prometheus.Metric) {
	statefulSets, err := w.statefulSetLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("List all statefulSet error:%v", err)
		return
	}
	for _, statefulSet := range statefulSets {
		statefulSetLabels, err := json.Marshal(statefulSet.Labels)
		if err != nil {
			klog.Errorf("Marshal daemonSet labels error:%v", err)
			continue
		}
		selector, err := metav1.LabelSelectorAsSelector(statefulSet.Spec.Selector)
		if err != nil {
			klog.Errorf("Parse deployment selector error:%v", err)
			return
		}

		containerNoneCareLabelValues := prometheus.Labels{
			values.WorkloadTypeLabelKey: "statefulset",
			values.WorkloadNameLabelKey: statefulSet.Name,
			values.NamespaceLabelKey:    statefulSet.Namespace,
			values.ClusterNameLabelKey:  w.clusterName,
			values.ClusterIdLabelKey:    w.clusterId,
			values.LabelsLabelKey:       string(statefulSetLabels),
		}
		w.collectStatefulSetPodMetrics(statefulSet, selector, containerNoneCareLabelValues, ch)
		w.collectStatefulSetCostMetrics(statefulSet, selector, containerNoneCareLabelValues, ch)

		containerCareLabelValues := prometheus.Labels{
			values.WorkloadTypeLabelKey: "statefulset",
			values.WorkloadNameLabelKey: statefulSet.Name,
			values.NamespaceLabelKey:    statefulSet.Namespace,
			values.ClusterNameLabelKey:  w.clusterName,
			values.ClusterIdLabelKey:    w.clusterId,
			values.LabelsLabelKey:       string(statefulSetLabels),
		}
		w.collectStatefulSetRequestMetrics(statefulSet, selector, containerCareLabelValues, ch)
		w.collectStatefulSetUsageMetrics(statefulSet, selector, containerCareLabelValues, ch)
	}
}

func (w *workloadMetricsCollector) collectStatefulSetPodMetrics(sf *appsv1.StatefulSet, selector labels.Selector,
	labels prometheus.Labels, ch chan<- prometheus.Metric) {
	pods, err := w.podLister.Pods(sf.Namespace).List(selector)
	if err != nil {
		klog.Errorf("List all pods error:%v", err)
		return
	}
	podCount := float64(len(pods))
	labels[values.ResourceTypeLabelKey] = "pod"
	ch <- prometheus.MustNewConstMetric(workloadPodCountDesc,
		prometheus.GaugeValue, podCount, utils.ConvertPrometheusLabelValuesInOrder(workloadCRNoneCareLabelKey, labels)...)
}

func (w *workloadMetricsCollector) collectStatefulSetRequestMetrics(sf *appsv1.StatefulSet, selector labels.Selector,
	labels prometheus.Labels, ch chan<- prometheus.Metric) {
	pods, err := w.podLister.Pods(sf.Namespace).List(selector)
	if err != nil {
		klog.Errorf("List all pods error:%v", err)
		return
	}

	cpuTotalRequest, ramTotalRequest := map[string]float64{}, map[string]float64{}
	for _, pod := range pods {
		cpu, ram := utils.ParsePodResourceRequest(pod.Spec.Containers)
		for containerName, value := range cpu {
			if _, ok := cpuTotalRequest[containerName]; !ok {
				cpuTotalRequest[containerName] = 0
			}
			cpuTotalRequest[containerName] += value
		}
		for containerName, value := range ram {
			if _, ok := ramTotalRequest[containerName]; !ok {
				ramTotalRequest[containerName] = 0
			}
			ramTotalRequest[containerName] += value
		}
	}

	labels[values.ResourceTypeLabelKey] = string(corev1.ResourceCPU)
	for containerName, cpu := range cpuTotalRequest {
		labels[values.ContainerNameLabelKey] = containerName
		ch <- prometheus.MustNewConstMetric(workloadResourceRequestDesc,
			prometheus.GaugeValue, cpu, utils.ConvertPrometheusLabelValuesInOrder(workloadCRCareLabelKey, labels)...)
	}
	labels[values.ResourceTypeLabelKey] = string(corev1.ResourceMemory)
	for containerName, ram := range ramTotalRequest {
		labels[values.ContainerNameLabelKey] = containerName
		ch <- prometheus.MustNewConstMetric(workloadResourceRequestDesc,
			prometheus.GaugeValue, ram, utils.ConvertPrometheusLabelValuesInOrder(workloadCRCareLabelKey, labels)...)
	}
}

func (w *workloadMetricsCollector) collectStatefulSetUsageMetrics(sf *appsv1.StatefulSet, selector labels.Selector,
	labels prometheus.Labels, ch chan<- prometheus.Metric) {
	pods, err := w.metricsClient.MetricsV1beta1().PodMetricses(sf.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		klog.Errorf("List all pod metrics error:%v, kubernetes metrics server may not be installed", err)
		return
	}
	cpuTotalUsage, memoryTotalUsage := map[string]float64{}, map[string]float64{}
	for _, pod := range pods.Items {
		cpu, ram := utils.ParsePodResourceUsage(pod.Containers)
		for containerName, value := range cpu {
			if _, ok := cpuTotalUsage[containerName]; !ok {
				cpuTotalUsage[containerName] = 0
			}
			cpuTotalUsage[containerName] += value
		}
		for containerName, value := range ram {
			if _, ok := memoryTotalUsage[containerName]; !ok {
				memoryTotalUsage[containerName] = 0
			}
			memoryTotalUsage[containerName] += value
		}
	}

	labels[values.ResourceTypeLabelKey] = string(corev1.ResourceCPU)
	for containerName, cpu := range cpuTotalUsage {
		labels[values.ContainerNameLabelKey] = containerName
		ch <- prometheus.MustNewConstMetric(workloadResourceUsageDesc,
			prometheus.GaugeValue, cpu, utils.ConvertPrometheusLabelValuesInOrder(workloadCRCareLabelKey, labels)...)
	}

	labels[values.ResourceTypeLabelKey] = string(corev1.ResourceMemory)
	for containerName, ram := range memoryTotalUsage {
		labels[values.ContainerNameLabelKey] = containerName
		ch <- prometheus.MustNewConstMetric(workloadResourceUsageDesc,
			prometheus.GaugeValue, ram, utils.ConvertPrometheusLabelValuesInOrder(workloadCRCareLabelKey, labels)...)
	}
}

func (w *workloadMetricsCollector) collectStatefulSetCostMetrics(sf *appsv1.StatefulSet, selector labels.Selector,
	labels prometheus.Labels, ch chan<- prometheus.Metric) {
	pods, err := w.podLister.Pods(sf.Namespace).List(selector)
	if err != nil {
		klog.Errorf("List all pods error:%v", err)
		return
	}

	totalCost := 0.0
	for _, pod := range pods {
		totalCost += utils.ParsePodResourceCost(pod, w.provider, w.nodeLister)
	}

	labels[values.ResourceTypeLabelKey] = "cost"
	ch <- prometheus.MustNewConstMetric(workloadResourceCostDesc,
		prometheus.GaugeValue, totalCost, utils.ConvertPrometheusLabelValuesInOrder(workloadCRNoneCareLabelKey, labels)...)
}

func (w *workloadMetricsCollector) collectDeploymentResourceMetrics(ch chan<- prometheus.Metric) {
	deployments, err := w.deploymentLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("List all deployment error:%v", err)
		return
	}
	for _, deployment := range deployments {
		deploymentLabels, err := json.Marshal(deployment.Labels)
		if err != nil {
			klog.Errorf("Marshal daemonSet labels error:%v", err)
			continue
		}
		selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
		if err != nil {
			klog.Errorf("Parse deployment selector error:%v", err)
			return
		}

		containerNoneCareLabelValues := prometheus.Labels{
			values.WorkloadTypeLabelKey: "deployment",
			values.WorkloadNameLabelKey: deployment.Name,
			values.NamespaceLabelKey:    deployment.Namespace,
			values.ClusterNameLabelKey:  w.clusterName,
			values.ClusterIdLabelKey:    w.clusterId,
			values.LabelsLabelKey:       string(deploymentLabels),
		}
		w.collectDeploymentPodMetrics(deployment, selector, containerNoneCareLabelValues, ch)
		w.collectDeploymentCostMetrics(deployment, selector, containerNoneCareLabelValues, ch)

		containerCareLabelValues := prometheus.Labels{
			values.WorkloadTypeLabelKey: "deployment",
			values.WorkloadNameLabelKey: deployment.Name,
			values.NamespaceLabelKey:    deployment.Namespace,
			values.ClusterNameLabelKey:  w.clusterName,
			values.ClusterIdLabelKey:    w.clusterId,
			values.LabelsLabelKey:       string(deploymentLabels),
		}
		w.collectDeploymentRequestMetrics(deployment, selector, containerCareLabelValues, ch)
		w.collectDeploymentUsageMetrics(deployment, selector, containerCareLabelValues, ch)
	}
}

func (w *workloadMetricsCollector) collectDeploymentPodMetrics(dm *appsv1.Deployment, selector labels.Selector,
	labels prometheus.Labels, ch chan<- prometheus.Metric) {
	pods, err := w.podLister.Pods(dm.Namespace).List(selector)
	if err != nil {
		klog.Errorf("List all pods error:%v", err)
		return
	}
	podCount := float64(len(pods))
	labels[values.ResourceTypeLabelKey] = "pod"
	ch <- prometheus.MustNewConstMetric(workloadPodCountDesc,
		prometheus.GaugeValue, podCount, utils.ConvertPrometheusLabelValuesInOrder(workloadCRNoneCareLabelKey, labels)...)
}

func (w *workloadMetricsCollector) collectDeploymentRequestMetrics(dm *appsv1.Deployment, selector labels.Selector,
	labels prometheus.Labels, ch chan<- prometheus.Metric) {
	pods, err := w.podLister.Pods(dm.Namespace).List(selector)
	if err != nil {
		klog.Errorf("List all pods error:%v", err)
		return
	}

	cpuTotalRequest, ramTotalRequest := map[string]float64{}, map[string]float64{}
	for _, pod := range pods {
		cpu, ram := utils.ParsePodResourceRequest(pod.Spec.Containers)
		for containerName, value := range cpu {
			if _, ok := cpuTotalRequest[containerName]; !ok {
				cpuTotalRequest[containerName] = 0
			}
			cpuTotalRequest[containerName] += value
		}
		for containerName, value := range ram {
			if _, ok := ramTotalRequest[containerName]; !ok {
				ramTotalRequest[containerName] = 0
			}
			ramTotalRequest[containerName] += value
		}
	}

	labels[values.ResourceTypeLabelKey] = string(corev1.ResourceCPU)
	for containerName, cpu := range cpuTotalRequest {
		labels[values.ContainerNameLabelKey] = containerName
		ch <- prometheus.MustNewConstMetric(workloadResourceRequestDesc,
			prometheus.GaugeValue, cpu, utils.ConvertPrometheusLabelValuesInOrder(workloadCRCareLabelKey, labels)...)
	}
	labels[values.ResourceTypeLabelKey] = string(corev1.ResourceMemory)
	for containerName, ram := range ramTotalRequest {
		labels[values.ContainerNameLabelKey] = containerName
		ch <- prometheus.MustNewConstMetric(workloadResourceRequestDesc,
			prometheus.GaugeValue, ram, utils.ConvertPrometheusLabelValuesInOrder(workloadCRCareLabelKey, labels)...)
	}
}

func (w *workloadMetricsCollector) collectDeploymentUsageMetrics(dm *appsv1.Deployment, selector labels.Selector,
	labels prometheus.Labels, ch chan<- prometheus.Metric) {
	pods, err := w.metricsClient.MetricsV1beta1().PodMetricses(dm.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		klog.Errorf("List all pod metrics error:%v, kubernetes metrics server may not be installed", err)
		return
	}
	cpuTotalUsage, memoryTotalUsage := map[string]float64{}, map[string]float64{}
	for _, pod := range pods.Items {
		cpu, ram := utils.ParsePodResourceUsage(pod.Containers)
		for containerName, value := range cpu {
			if _, ok := cpuTotalUsage[containerName]; !ok {
				cpuTotalUsage[containerName] = 0
			}
			cpuTotalUsage[containerName] += value
		}
		for containerName, value := range ram {
			if _, ok := memoryTotalUsage[containerName]; !ok {
				memoryTotalUsage[containerName] = 0
			}
			memoryTotalUsage[containerName] += value
		}
	}

	labels[values.ResourceTypeLabelKey] = string(corev1.ResourceCPU)
	for containerName, cpu := range cpuTotalUsage {
		labels[values.ContainerNameLabelKey] = containerName
		ch <- prometheus.MustNewConstMetric(workloadResourceUsageDesc,
			prometheus.GaugeValue, cpu, utils.ConvertPrometheusLabelValuesInOrder(workloadCRCareLabelKey, labels)...)
	}

	labels[values.ResourceTypeLabelKey] = string(corev1.ResourceMemory)
	for containerName, ram := range memoryTotalUsage {
		labels[values.ContainerNameLabelKey] = containerName
		ch <- prometheus.MustNewConstMetric(workloadResourceUsageDesc,
			prometheus.GaugeValue, ram, utils.ConvertPrometheusLabelValuesInOrder(workloadCRCareLabelKey, labels)...)
	}
}

func (w *workloadMetricsCollector) collectDeploymentCostMetrics(dm *appsv1.Deployment, selector labels.Selector,
	labels prometheus.Labels, ch chan<- prometheus.Metric) {
	pods, err := w.podLister.Pods(dm.Namespace).List(selector)
	if err != nil {
		klog.Errorf("List all pods error:%v", err)
		return
	}

	totalCost := 0.0
	for _, pod := range pods {
		totalCost += utils.ParsePodResourceCost(pod, w.provider, w.nodeLister)
	}

	labels[values.ResourceTypeLabelKey] = "cost"
	ch <- prometheus.MustNewConstMetric(workloadResourceCostDesc,
		prometheus.GaugeValue, totalCost, utils.ConvertPrometheusLabelValuesInOrder(workloadCRNoneCareLabelKey, labels)...)
}

func RegisterWorkloadLevelMetricsCollection(agentOptions *options.AgentOptions,
	client *versioned.Clientset,
	provider cloudprice.CloudProviderInterface,
	coreResourceInformerLister *api.CoreResourceInformerLister) {
	workloadMetricsCollector := &workloadMetricsCollector{
		clusterId:         agentOptions.ClusterId,
		clusterName:       agentOptions.ClusterName,
		metricsClient:     client,
		provider:          provider,
		podLister:         coreResourceInformerLister.PodLister,
		nodeLister:        coreResourceInformerLister.NodeLister,
		deploymentLister:  coreResourceInformerLister.DeploymentLister,
		statefulSetLister: coreResourceInformerLister.StatefulSetLister,
		daemonSetLister:   coreResourceInformerLister.DaemonSetLister,
	}

	prometheus.MustRegister(workloadMetricsCollector)
}
