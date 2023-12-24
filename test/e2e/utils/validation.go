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
	"kubefin.dev/kubefin/pkg/analyzer/types"
)

func ValidateAllClustersMetricsSummary(clustersSummary *types.ClusterResourcesSummaryList) bool {
	for _, cluster := range clustersSummary.Items {
		if cluster.ClusterConnectionSate != "running" ||
			cluster.NodeNumbersCurrent == 0 ||
			cluster.PodTotalCurrent == 0 ||
			cluster.PodScheduledCurrent == 0 ||
			cluster.CPUCoreTotal == 0 ||
			cluster.RAMGiBTotal == 0 {
			return false
		}
	}

	return true
}

func ValidateSpecificClusterMetricsSummary(clusterSummary *types.ClusterResourcesSummary) bool {
	if clusterSummary.ClusterConnectionSate != "running" ||
		clusterSummary.NodeNumbersCurrent == 0 ||
		clusterSummary.PodTotalCurrent == 0 ||
		clusterSummary.PodScheduledCurrent == 0 ||
		clusterSummary.CPUCoreTotal == 0 ||
		clusterSummary.RAMGiBTotal == 0 {
		return false
	}

	return true
}

func ValidateAllClustersCostsSummary(clustersSummary *types.ClusterCostsSummaryList) bool {
	for _, cluster := range clustersSummary.Items {
		if cluster.ClusterConnectionSate != "running" ||
			cluster.ClusterMonthCostCurrent == 0 ||
			cluster.ClusterMonthEstimateCost == 0 ||
			cluster.ClusterAvgDailyCost == 0 ||
			cluster.ClusterAvgHourlyCoreCost == 0 {
			return false
		}
	}

	return true
}

func ValidateSpecificClusterCostsSummary(clusterSummary *types.ClusterCostsSummary) bool {
	if clusterSummary.ClusterConnectionSate != "running" ||
		clusterSummary.ClusterMonthCostCurrent == 0 ||
		clusterSummary.ClusterMonthEstimateCost == 0 ||
		clusterSummary.ClusterAvgDailyCost == 0 ||
		clusterSummary.ClusterAvgHourlyCoreCost == 0 {
		return false
	}

	return true
}
