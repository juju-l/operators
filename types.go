package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HelmSpec struct {
	ReleaseName string `json:"releaseName"`
	Namespace   string `json:"namespace"`
	Chart       string `json:"chart"`
	Version     string `json:"version,omitempty"`
	Values      string `json:"values,omitempty"`
}

type HelmStatus struct {
	Phase   string `json:"phase,omitempty"`
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type Helm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelmSpec   `json:"spec,omitempty"`
	Status HelmStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type HelmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Helm `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Helm{}, &HelmList{})
}