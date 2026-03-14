package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// Tst 是自定义资源，Namespaced 作用域
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Tst struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TstSpec   `json:"spec"`
	Status TstStatus `json:"status"`
}

// TstSpec 期望状态
type TstSpec struct {
	Replicas int32  `json:"replicas"`
	Image    string `json:"image"`
}

// TstStatus 实际状态
type TstStatus struct {
	ReadyReplicas int32  `json:"readyReplicas"`
	State         string `json:"state"` // Active/Error
	Message       string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TstList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tst `json:"items"`
}