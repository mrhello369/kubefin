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

package resources_handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/kubefin/kubefin/pkg/api"
	"github.com/kubefin/kubefin/pkg/server/implementation"
	"github.com/kubefin/kubefin/pkg/utils"
)

// ClusterCPUResourcesHandler      godoc
//
//	@Summary		Get specific cluster CPU resources metrics
//	@Description	Get specific cluster CPU resources metrics
//	@Tags			Resources
//	@Produce		json
//	@Param			cluster_id	path		string	true	"Cluster Id"
//	@Param			startTime	query		uint64	false	"The start time to query"
//	@Param			endTime		query		uint64	false	"The end time to query"
//	@Param			stepSeconds	query		uint64	false	"The step seconds of the data to return"
//	@Success		200			{object}	api.ClusterResourceMetrics
//	@Failure		500			{object}	api.StatusError
//	@Router			/resources/clusters/{cluster_id}/cpu [get]
func ClusterCPUResourcesHandler(ctx *gin.Context) {
	klog.Info("Start to query cluster CPU resources metrics")
	clusterResourcesHandler(ctx, v1.ResourceCPU)
}

// ClusterMemoryResourcesHandler   godoc
//
//	@Summary		Get specific cluster memory resources metrics
//	@Description	Get specific cluster memory resources metrics
//	@Tags			Resources
//	@Produce		json
//	@Param			cluster_id	path		string	true	"Cluster Id"
//	@Param			startTime	query		uint64	false	"The start time to query"
//	@Param			endTime		query		uint64	false	"The end time to query"
//	@Param			stepSeconds	query		uint64	false	"The step seconds of the data to return"
//	@Success		200			{object}	api.ClusterResourceMetrics
//	@Failure		500			{object}	api.StatusError
//	@Router			/resources/clusters/{cluster_id}/memory [get]
func ClusterMemoryResourcesHandler(ctx *gin.Context) {
	klog.Info("Start to query cluster Memory resources metrics")
	clusterResourcesHandler(ctx, v1.ResourceMemory)
}

func clusterResourcesHandler(ctx *gin.Context, resourceType v1.ResourceName) {
	tenantId := utils.ParserTenantIdFromCtx(ctx)
	clusterId := utils.ParseClusterFromCtx(ctx)
	if clusterId == "" {
		utils.ForwardStatusError(ctx, http.StatusBadRequest,
			api.QueryParaErrorStatus, api.QueryParaErrorReason, "")
		return
	}
	startTime, endTime, stepSeconds, err := utils.GetClusterDetailsStartEndStepTime(tenantId, clusterId)
	if err != nil {
		utils.ForwardStatusError(ctx, http.StatusInternalServerError,
			api.QueryFailedStatus, api.QueryFailedReason, err.Error())
		return
	}
	clusterMemoryMetrics, err := implementation.QueryClusterResourcesSummaryWithTimeRange(tenantId, clusterId, resourceType, startTime, endTime, stepSeconds)
	if err != nil {
		utils.ForwardStatusError(ctx, http.StatusInternalServerError,
			api.QueryFailedStatus, api.QueryFailedReason, err.Error())
		return
	}
	if clusterMemoryMetrics == nil {
		msg := fmt.Sprintf("Resource(%s) not found for cluster(%s) from %d to %d",
			resourceType, clusterId, startTime, endTime)
		utils.ForwardStatusError(ctx, http.StatusNotFound,
			api.QueryNotFoundStatus, api.QueryNotFoundReason, msg)
		return
	}

	bodyBytes, err := json.Marshal(clusterMemoryMetrics)
	if err != nil {
		utils.ForwardStatusError(ctx, http.StatusInternalServerError,
			api.QueryFailedStatus, api.QueryFailedReason, err.Error())
		return
	}

	ctx.Data(http.StatusOK, "application/json", bodyBytes)
}
