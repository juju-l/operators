package helms

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// AddToScheme 注册到 client-go Scheme
func AddToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion, &Hlm{}, &HlmList{})
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
