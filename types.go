package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion CRD 组和版本
var SchemeGroupVersion = schema.GroupVersion{
	Group:   "tpl.vipex.cc",
	Version: "v1alpha1",
}

// KindHlm CRD Kind
const KindHlm = "Hlm"

// Hlm 自定义资源
type Hlm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HlmSpec   `json:"spec"`
	Status HlmStatus `json:"status,omitempty"`
}

// HlmSpec 仅管理 valuesYAML
type HlmSpec struct {
	ReleaseName string `json:"releaseName"` // Helm Release 名
	Namespace   string `json:"namespace"`   // 命名空间
	ValuesYAML  string `json:"valuesYAML"`  // 自定义 values
}

// HlmStatus 状态
type HlmStatus struct {
	Phase      string      `json:"phase,omitempty"`
	ReleaseRef string      `json:"releaseRef,omitempty"`
	Conditions []Condition `json:"conditions,omitempty"`
}

// Condition 状态条件
type Condition struct {
	Type               string      `json:"type"`
	Status             string      `json:"status"`
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

// DeepCopyObject 实现 runtime.Object
func (in *Hlm) DeepCopyObject() runtime.Object {
	out := &Hlm{}
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = *in.ObjectMeta.DeepCopy()
	out.Spec = in.Spec
	out.Status = in.Status
	return out
}

// HlmList 列表
type HlmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Hlm `json:"items"`
}

// DeepCopyObject 实现 runtime.Object
func (in *HlmList) DeepCopyObject() runtime.Object {
	out := &HlmList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	out.Items = make([]Hlm, len(in.Items))
	copy(out.Items, in.Items)
	return out
}
