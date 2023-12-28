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

package types

type InstancePriceInfo struct {
	// NodeTotalHourlyPrice represents the total cost of the node in one hour
	NodeTotalHourlyPrice float64
	// CPUCore represents the total cpu cores of the node
	CPUCore float64
	// CPUCoreHourlyPrice represents the one core hourly cost of the node
	CPUCoreHourlyPrice float64
	// RamGiB represents the total memory GiB of the node
	RamGiB float64
	// RAMGiBHourlyPrice represents the one GiB hourly cost of the node
	RAMGiBHourlyPrice float64
	// GPUCards represents the total GPU cards of the node
	GPUCards float64
	// GPUCardHourlyPrice represents the one GPU card hourly cost of the node
	GPUCardHourlyPrice float64
	// InstanceType represents the type of the node, such as t3.2xlarge
	InstanceType string
	// InstanceType represents the billing mode of the node, such as OnDemand
	BillingMode string
	// BillingPeriod only needs for monthly/yearly billing mode
	BillingPeriod int
	// Region represents the node's region
	Region string
	// CloudProvider represents the node's cloud provider
	CloudProvider string
}
