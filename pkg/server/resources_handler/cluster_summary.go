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
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/kubefin/kubefin/pkg/api"
	implementation "github.com/kubefin/kubefin/pkg/server/implementation"
	"github.com/kubefin/kubefin/pkg/utils"
)

// ClustersResourcesSummaryHandler godoc
//
//	@Summary		Get all clusters resources summary
//	@Description	Get all clusters resources summary in current two month
//	@Tags			resources
//	@Produce		json
//	@Success		200	{object}	api.ClusterResourcesSummaryList
//	@Failure		500	{object}	api.StatusError
//	@Router			/resources/summary [get]
func ClustersResourcesSummaryHandler(ctx *gin.Context) {
	klog.Infof("Start to query clusters metrics summary")
	tenantId := utils.ParserTenantIdFromCtx(ctx)
	// If data not comes up in two-month period, we will ignore it
	start, end := utils.GetCurrentTwoMonthStartEndTime()
	allClustersProperty, err := implementation.QueryAllClustersBasicProperty(tenantId, start, end)
	if err != nil {
		utils.ForwardStatusError(ctx, http.StatusInternalServerError,
			api.QueryFailedStatus, api.QueryFailedReason, err.Error())
		return
	}
	allClustersSummary, err := implementation.QueryAllClustersCurrentMetrics(tenantId)
	if err != nil {
		utils.ForwardStatusError(ctx, http.StatusInternalServerError,
			api.QueryFailedStatus, api.QueryFailedReason, err.Error())
		return
	}
	summaries := implementation.ConvertToMultiClustersResourcesList(allClustersSummary, allClustersProperty)
	bodyBytes, err := json.Marshal(summaries)
	if err != nil {
		utils.ForwardStatusError(ctx, http.StatusInternalServerError,
			api.QueryFailedStatus, api.QueryFailedReason, err.Error())
		return
	}
	ctx.Data(http.StatusOK, "application/json", bodyBytes)
}

// ClusterResourcesSummaryHandler  godoc
//
//	@Summary		Get specific cluster resources summary
//	@Description	Get specific cluster resources summary in current two month
//	@Tags			Resources
//	@Produce		json
//	@Param			cluster_id	path		string	true	"Cluster Id"
//	@Success		200			{object}	api.ClusterResourcesSummary
//	@Failure		500			{object}	api.StatusError
//	@Router			/resources/clusters/{cluster_id}/summary [get]
func ClusterResourcesSummaryHandler(ctx *gin.Context) {
	klog.Infof("Start to query specific cluster metrics summary")
	clusterId := utils.ParseClusterFromCtx(ctx)
	tenantId := utils.ParserTenantIdFromCtx(ctx)
	// If data not comes up in two-month period, we will ignore it
	start, end := utils.GetCurrentTwoMonthStartEndTime()
	clustersProperty, err := implementation.QueryClusterBasicProperty(tenantId, clusterId, start, end)
	if err != nil {
		utils.ForwardStatusError(ctx, http.StatusInternalServerError,
			api.QueryFailedStatus, api.QueryFailedReason, err.Error())
		return
	}
	clusterSummary, err := implementation.QueryClusterCurrentResources(tenantId, clusterId)
	if err != nil {
		utils.ForwardStatusError(ctx, http.StatusInternalServerError,
			api.QueryFailedStatus, api.QueryFailedReason, err.Error())
		return
	}
	clusterSummary.ClusterBasicProperty = *clustersProperty

	bodyBytes, err := json.Marshal(clusterSummary)
	if err != nil {
		utils.ForwardStatusError(ctx, http.StatusInternalServerError,
			api.QueryFailedStatus, api.QueryFailedReason, err.Error())
		return
	}
	ctx.Data(http.StatusOK, "application/json", bodyBytes)
}
