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

package metrics

import (
	"context"

	"k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/kubefin/kubefin/cmd/kubefin-agent/app/options"
	"github.com/kubefin/kubefin/pkg/api"
	"github.com/kubefin/kubefin/pkg/cloudprice"
	"github.com/kubefin/kubefin/pkg/metrics/core"
)

func RegisterAgentMetricsCollector(ctx context.Context,
	options *options.AgentOptions,
	coreResourceInformerLister *api.CoreResourceInformerLister,
	provider cloudprice.CloudProviderInterface,
	metricsClientSet *versioned.Clientset) {
	core.RegisterClusterLevelMetricsCollection(options, metricsClientSet, provider, coreResourceInformerLister)
	core.RegisterPodLevelMetricsCollection(options, metricsClientSet, provider, coreResourceInformerLister)
	core.RegisterNodeLevelMetricsCollection(options, metricsClientSet, provider, coreResourceInformerLister)
	core.RegisterWorkloadLevelMetricsCollection(options, metricsClientSet, provider, coreResourceInformerLister)
}
