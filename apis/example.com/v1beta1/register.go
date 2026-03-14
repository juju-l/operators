package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// 【新增 client-gen 标记】指定组名和版本
// +k8s:groupName=example.com
var SchemeGroupVersion = schema.GroupVersion{
	Group:   "example.com",
	Version: "v1beta1",
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Tst{},
		&TstList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}