package hlm

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const GroupName = "example.com"
const GroupVersion = "v1beta1"

var SchemeGroupVersion = schema.GroupVersion{
	Group:   GroupName,
	Version: GroupVersion,
}

var SchemeGroupVersionResource = schema.GroupVersionResource{
	Group:    GroupName,
	Version:  GroupVersion,
	Resource: "hlms",
}

func AddToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Hlm{},
		&HlmList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}