/*
Copyright 2024.

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

package v1

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ScanFrequencyPolicy string

const (
	OneOff    ScanFrequencyPolicy = "OneOff"
	Recurring ScanFrequencyPolicy = "Recurring"
)

type ScanStatus string

const (
	ScanSuccess ScanStatus = "success"
	ScanRunning ScanStatus = "running"
	ScanFailed  ScanStatus = "failed"
)

// ClusterScanSpec defines the desired state of ClusterScan
type ClusterScanSpec struct {
	//+kubebuilder:default=terrascan
	ScanType string `json:"scanType,omitempty"`

	//+kubebuilder:validation:MinLength=0
	Schedule string `json:"schedule,omitempty"`

	//+optional
	ScanFrequencyPolicy ScanFrequencyPolicy `json:"scanFrequency,omitempty"`

	//+optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// ClusterScanStatus defines the observed state of ClusterScan
type ClusterScanStatus struct {
	//+optional
	Active []corev1.ObjectReference `json:"active,omitempty"`

	//+optional
	LastScanTime *metav1.Time `json:"lastScanTime,omitempty"`

	//+optional
	ScanStatus ScanStatus `json:"scanStatus,omitempty"`

	//+optional
	Result json.RawMessage `json:"result,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClusterScan is the Schema for the clusterscans API
type ClusterScan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterScanSpec   `json:"spec,omitempty"`
	Status ClusterScanStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterScanList contains a list of ClusterScan
type ClusterScanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterScan `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterScan{}, &ClusterScanList{})
}
