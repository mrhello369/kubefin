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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Cluster,shortName=cac,categories={kubefin-dev}
// +kubebuilder:metadata:labels=kubefin.dev/crd-install=true

// CustomAllocationConfiguration represents the custom allocation insight configuration.
// It could be used to view the cost/resource allocation of various resources.
type CustomAllocationConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents how to collect different CR resource/cost allocation metrics
	// +required
	Spec CustomAllocationConfigurationSpec `json:"spec"`
}

type CustomAllocationConfigurationSpec struct {
	// WorkloadsAllocation represents the list of workload allocation configurations
	// +required
	// +kubebuilder:validation:MinItems=1
	WorkloadsAllocation []WorkloadAllocation `json:"workloadsAllocation"`
}

type WorkloadAllocation struct {
	// WorkloadTypeAlias represents the alias of the workload type
	// +required
	WorkloadTypeAlias string `json:"workloadTypeAlias"`
	// Target represents the target resources kind of the workload
	// +required
	Target WorkloadAllocationTarget `json:"target"`
	// PodLabelSelectorExtract represents the way used to extract the labels to
	// find the corresponding pods of the workload
	// +required
	PodLabelSelectorExtract PodLabelSelectorExtract `json:"podLabelSelectorExtract"`
}

type WorkloadAllocationTarget struct {
	// APIVersion represents the API version of the target resources.
	// +required
	APIVersion string `json:"apiVersion"`

	// Kind represents the Kind of the target resources.
	// +required
	Kind string `json:"kind"`
}

type PodLabelSelectorExtract struct {
	// Script represents the lua script used to extract the labels to find the corresponding pods of the workload
	// +required
	Script string `json:"script"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CustomAllocationConfigurationList contains a list of CustomAllocationConfiguration.
type CustomAllocationConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomAllocationConfiguration `json:"items"`
}
