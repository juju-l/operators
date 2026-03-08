package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HlmSpec defines the desired state of Hlm
type HlmSpec struct {
	// Chart is the name of the chart inside the provided helm package or repo
	Chart string `json:"chart,omitempty"`
	// Values to pass to Helm as a YAML/JSON map (fallback generic values)
	Values map[string]interface{} `json:"values,omitempty"`
	// ReleaseName is the name to use for the helm release
	ReleaseName string `json:"releaseName,omitempty"`
	// Namespace where to install the release
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// Fields derived from _.valuecustomtpls.yml
	Env string `json:"env,omitempty"`
	StsSingle *bool `json:"stsSingle,omitempty"`
	App string `json:"app,omitempty"`
	CloudNS string `json:"cloudns,omitempty"`
	// Helmv3tpl contains environment keyed helm values (e.g. helmv3tpl.devcn.app07...)
	Helmv3tpl map[string]interface{} `json:"helmv3tpl,omitempty"`
	Coredns string `json:"coredns,omitempty"`
	// Global contains global settings (reg, label, tag etc.)
	Global map[string]interface{} `json:"global,omitempty"`
}

// HlmStatus defines the observed state of Hlm
type HlmStatus struct {
	// LastReconcile is the last time operator reconciled this resource
	LastReconcile metav1.Time `json:"lastReconcile,omitempty"`
	// Conditions provide status details
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastAction is the last performed action (Install|Upgrade|Uninstall|Rollback|Template)
	LastAction string `json:"lastAction,omitempty"`
	// LastActionStatus is the result of the last action (Succeeded|Failed)
	LastActionStatus string `json:"lastActionStatus,omitempty"`
	// LastActionMessage is an optional message or error description
	LastActionMessage string `json:"lastActionMessage,omitempty"`
	// LastRevision records last known release revision if available
	LastRevision int `json:"lastRevision,omitempty"`
	// RenderedManifestConfigMap stores the name of the ConfigMap that holds the last rendered manifest
	RenderedManifestConfigMap string `json:"renderedManifestConfigMap,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Hlm is the Schema for the hlms API
type Hlm struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HlmSpec   `json:"spec,omitempty"`
	Status HlmStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HlmList contains a list of Hlm
type HlmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Hlm `json:"items"`
}
