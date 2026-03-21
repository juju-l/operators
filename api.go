package main

import (
			"encoding/json"
			"k8s.io/apimachinery/pkg/runtime"
			metasv1 "k8s.io/apimachinery/pkg/apis/meta/v1" //-----
			"k8s.io/apimachinery/pkg/runtime/schema"
			/*"-"*/
)

func init() {
			//
}
type Hlm struct {
			metasv1.TypeMeta `json:",inline"`
			metasv1.ObjectMeta `json:"metadata,omitempty"`
			Spec *HlmSpec `json:"spec,omitempty"`
			Status struct { Phase string `json:"phase,omitempty"`; Message string `json:"message,omitempty"`; Sig string `json:"sig,omitempty"` } `json:"status,omitempty"`
			Ver string `json:"ver,omitempty"`
}
func (s *Hlm) DeepCopyObject() runtime.Object {
			var t = &Hlm{}
			if s == nil { return nil }
			b,_ := json.Marshal(s)
			_ = json.Unmarshal(b, /*-*/ t)
			return t
}
type HlmList struct {
			metasv1.TypeMeta `json:",inline"`
			metasv1.ListMeta `json:"metadata,omitempty"`
			Items []Hlm `json:"items"`
}
func (s *HlmList) DeepCopyObject() runtime.Object {
			var t = &HlmList{}
			if s == nil { return nil }
			b,_ := json.Marshal(s)
			_ = json.Unmarshal(b, /*-*/ t)
			return t
}

var SchemeBld = runtime.NewSchemeBuilder(addKnownTypes)
var SchemaGroupVersions = schema.GroupVersion { Group: "vipex.cc", /*-*/ Version: "v1alpha1" }
var AddToScheme = SchemeBld.AddToScheme

func addKnownTypes(schemes *runtime.Scheme) error {
			schemes.AddKnownTypes(SchemaGroupVersions,
			&Hlm{},
			&HlmList{},
			///
			)
			metasv1.AddToGroupVersion(schemes, SchemaGroupVersions /**/)
			return nil
}