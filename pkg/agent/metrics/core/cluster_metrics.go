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

package core

import (
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/labels"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"kubefin.dev/kubefin/cmd/kubefin-agent/app/options"
	"kubefin.dev/kubefin/pkg/agent/cloudprice"
	"kubefin.dev/kubefin/pkg/agent/utils"
	"kubefin.dev/kubefin/pkg/values"
)

var (
	clusterMetricsLabelKey = []string{
		values.RegionLabelKey,
		values.CloudProviderLabelKey,
		values.ClusterNameLabelKey,
		values.ClusterIdLabelKey,
	}

	clusterStateDesc = prometheus.NewDesc(
		values.ClusterActiveMetricsName,
		"The cluster state metrics", clusterMetricsLabelKey, nil)
)

type clusterStateMetricsCollector struct {
	cloudProvider string
	clusterName   string
	clusterId     string

	provider   cloudprice.CloudProviderInterface
	nodeLister v1.NodeLister
}

func (c *clusterStateMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- clusterStateDesc
}

func (c *clusterStateMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	nodes, err := c.nodeLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("List all nodes error:%v", err)
		return
	}

	nodeCostInfo, err := c.provider.GetNodeHourlyPrice(nodes[0])
	if err != nil {
		klog.Errorf("Get node hourly price error:%v", err)
		return
	}
	labels := prometheus.Labels{
		values.RegionLabelKey:        nodeCostInfo.Region,
		values.CloudProviderLabelKey: c.cloudProvider,
		values.ClusterNameLabelKey:   c.clusterName,
		values.ClusterIdLabelKey:     c.clusterId,
	}

	ch <- prometheus.MustNewConstMetric(clusterStateDesc,
		prometheus.GaugeValue, 1, utils.ConvertPrometheusLabelValuesInOrder(clusterMetricsLabelKey, labels)...)
}

func RegisterClusterLevelMetricsCollection(agentOptions *options.AgentOptions,
	provider cloudprice.CloudProviderInterface,
	coreResourceInformerLister *options.CoreResourceInformerLister) {
	clusterStateMetricsCollector := &clusterStateMetricsCollector{
		cloudProvider: agentOptions.CloudProvider,
		clusterId:     agentOptions.ClusterId,
		clusterName:   agentOptions.ClusterName,
		provider:      provider,
		nodeLister:    coreResourceInformerLister.NodeLister,
	}

	prometheus.MustRegister(clusterStateMetricsCollector)
}
