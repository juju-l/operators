package hlm

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Hlm
type Hlm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HlmSpec   `json:"spec"`
	Status HlmStatus `json:"status"`
}

type HlmSpec struct {
	// 你自己填字段
}

type HlmStatus struct {
	Ready  bool   `json:"ready"`
	Phase  string `json:"phase"`
	Reason string `json:"reason,omitempty"`
}

// HlmList
type HlmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Hlm `json:"items"`
}

// DeepCopyObject
func (in *Hlm) DeepCopyObject() runtime.Object {
	out := &Hlm{}
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = *in.ObjectMeta.DeepCopy()
	out.Spec = in.Spec
	out.Status = in.Status
	return out
}

func (in *HlmList) DeepCopyObject() runtime.Object {
	out := &HlmList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = *in.ListMeta.DeepCopy()
	out.Items = make([]Hlm, len(in.Items))
	for i := range in.Items {
		in.Items[i].DeepCopyObject().(*Hlm)
		out.Items[i] = in.Items[i]
	}
	return out
}